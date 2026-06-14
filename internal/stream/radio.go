package stream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// radioBufSize is the chunk size used when copying audio bytes to the client.
const radioBufSize = 16 * 1024

// probeTimeout bounds the short metadata/homepage requests made by ProbeStation.
const probeTimeout = 8 * time.Second

// StationInfo is the metadata discovered from a stream's ICY headers and
// (best-effort) its homepage. Empty fields mean "not detected".
type StationInfo struct {
	Name     string `json:"name"`
	Genre    string `json:"genre"`
	Homepage string `json:"homepage"`
	Favicon  string `json:"favicon"`
	Bitrate  int    `json:"bitrate"`
}

// ProbeStation opens a short connection to the stream, reads its ICY headers
// (icy-name/genre/url/br) without consuming audio, then tries to resolve a logo
// from the station homepage (og:image, falling back to /favicon.ico). It is
// best-effort: any field that can't be detected is left empty.
func ProbeStation(ctx context.Context, streamURL string) (StationInfo, error) {
	var info StationInfo

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Icy-MetaData", "1")
	req.Header.Set("User-Agent", "Timbre/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return info, err
	}
	// Read headers only; never drain the audio body.
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	info.Name = strings.TrimSpace(resp.Header.Get("icy-name"))
	info.Genre = strings.TrimSpace(resp.Header.Get("icy-genre"))
	info.Homepage = strings.TrimSpace(resp.Header.Get("icy-url"))
	info.Bitrate, _ = strconv.Atoi(resp.Header.Get("icy-br"))

	if info.Homepage != "" {
		info.Favicon = fetchLogo(ctx, info.Homepage)
	}
	return info, nil
}

var ogImageRe = regexp.MustCompile(`(?is)<meta[^>]+>`)

// fetchLogo fetches the homepage HTML and returns an absolute logo URL from an
// og:image / twitter:image meta tag, falling back to the site's /favicon.ico.
// Returns "" if the homepage can't be fetched.
func fetchLogo(ctx context.Context, homepage string) string {
	base, err := url.Parse(homepage)
	if err != nil || base.Host == "" {
		return ""
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, homepage, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Timbre/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return faviconFallback(base)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return faviconFallback(base)
	}

	// Cap the read: meta tags live in <head>.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if img := extractOGImage(string(body)); img != "" {
		if abs, err := base.Parse(img); err == nil {
			return abs.String()
		}
	}
	return faviconFallback(base)
}

func faviconFallback(base *url.URL) string {
	return base.Scheme + "://" + base.Host + "/favicon.ico"
}

// extractOGImage scans meta tags for og:image (preferred) or twitter:image and
// returns the content URL, or "" if none present.
func extractOGImage(html string) string {
	var twitter string
	for _, tag := range ogImageRe.FindAllString(html, -1) {
		lower := strings.ToLower(tag)
		if !strings.Contains(lower, "og:image") && !strings.Contains(lower, "twitter:image") {
			continue
		}
		content := metaContent(tag)
		if content == "" {
			continue
		}
		if strings.Contains(lower, "og:image") {
			return content // og:image wins immediately
		}
		if twitter == "" {
			twitter = content
		}
	}
	return twitter
}

var contentAttrRe = regexp.MustCompile(`(?is)content\s*=\s*["']([^"']+)["']`)

func metaContent(tag string) string {
	if m := contentAttrRe.FindStringSubmatch(tag); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// ProxyRadio dials the upstream web-radio URL and pipes its audio to w. It
// requests ICY (Shoutcast/Icecast) metadata; when the upstream interleaves
// metadata blocks into the byte stream, those blocks are stripped out (never
// forwarded to the client) and the parsed StreamTitle is delivered via onTitle
// on every change. The upstream connection is bound to ctx, so cancelling ctx
// (e.g. when the browser disconnects) tears it down.
//
// w must expose http.Flusher (echo's response writer does) so audio starts
// playing without waiting for a full buffer.
//
// When transcode is true the audio is re-encoded to MP3 via ffmpeg before being
// sent to the client. This guarantees playback in browsers that can't decode
// the upstream codec (e.g. raw AAC/ADTS or AAC+ live streams) at the cost of
// CPU. ICY metadata is still demuxed and reported via onTitle either way.
func ProxyRadio(ctx context.Context, w http.ResponseWriter, url string, transcode bool, onTitle func(string)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	// Ask for inline metadata; identify as a player some servers gate on.
	req.Header.Set("Icy-MetaData", "1")
	req.Header.Set("User-Agent", "Timbre/1.0")

	// No client timeout: radio streams are long-lived. Cancellation is driven
	// by ctx instead.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	metaInt, _ := strconv.Atoi(resp.Header.Get("icy-metaint"))

	if transcode {
		return transcodeRadio(ctx, w, resp.Body, metaInt, onTitle)
	}

	// Forward a sensible content type so the browser's <audio> can decode.
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "audio/mpeg"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)

	if metaInt <= 0 {
		// No interleaved metadata: straight passthrough.
		return copyFlush(w, resp.Body, flusher)
	}

	return demuxICY(w, resp.Body, metaInt, flusher, onTitle)
}

// flushWriter flushes the underlying http response after every write so
// transcoded audio reaches the client immediately.
type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func (fw flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return n, err
}

// transcodeRadio pipes the upstream audio through ffmpeg, re-encoding to MP3.
// ICY metadata (if metaInt > 0) is stripped from the upstream bytes before they
// reach ffmpeg and reported via onTitle, so now-playing still works.
func transcodeRadio(ctx context.Context, w http.ResponseWriter, body io.Reader, metaInt int, onTitle func(string)) error {
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", "pipe:0",
		"-vn",
		"-c:a", "libmp3lame", "-b:a", "128k", "-f", "mp3",
		"pipe:1",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	cmd.Stdout = flushWriter{w: w, f: flusher}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	// Feed the (metadata-stripped) upstream audio into ffmpeg's stdin.
	go func() {
		defer stdin.Close()
		if metaInt > 0 {
			_ = demuxICY(stdin, body, metaInt, nil, onTitle)
		} else {
			_, _ = io.Copy(stdin, body)
		}
	}()

	_ = cmd.Wait() // client disconnect cancels via ctx
	return nil
}

// copyFlush copies src to dst in radioBufSize chunks, flushing after each.
func copyFlush(dst io.Writer, src io.Reader, flusher http.Flusher) error {
	buf := make([]byte, radioBufSize)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// demuxICY reads an ICY stream where every metaInt audio bytes are followed by
// one length byte L and L*16 metadata bytes. Audio bytes are forwarded; the
// metadata block is parsed for StreamTitle and dropped from the output.
func demuxICY(w io.Writer, body io.Reader, metaInt int, flusher http.Flusher, onTitle func(string)) error {
	r := bufio.NewReaderSize(body, radioBufSize)
	audio := make([]byte, 0, metaInt)
	if cap(audio) < metaInt {
		audio = make([]byte, metaInt)
	}
	audio = audio[:metaInt]

	var lastTitle string
	for {
		// Read exactly metaInt audio bytes and forward them.
		if _, err := io.ReadFull(r, audio); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}
		if _, err := w.Write(audio); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}

		// One length byte: metadata length is lenByte*16.
		lenByte, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		metaLen := int(lenByte) * 16
		if metaLen == 0 {
			continue // no metadata in this block
		}

		meta := make([]byte, metaLen)
		if _, err := io.ReadFull(r, meta); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}
		if title := parseStreamTitle(meta); title != "" && title != lastTitle {
			lastTitle = title
			if onTitle != nil {
				onTitle(title)
			}
		}
	}
}

// parseStreamTitle extracts the StreamTitle value from an ICY metadata block of
// the form: StreamTitle='Artist - Song';StreamUrl='...'; Trailing NUL padding
// and surrounding quotes are removed.
func parseStreamTitle(meta []byte) string {
	s := string(meta)
	const key = "StreamTitle='"
	i := strings.Index(s, key)
	if i < 0 {
		return ""
	}
	s = s[i+len(key):]
	// Value ends at the closing "';" (or the last quote if malformed).
	if j := strings.Index(s, "';"); j >= 0 {
		s = s[:j]
	} else if j := strings.IndexByte(s, '\''); j >= 0 {
		s = s[:j]
	}
	return strings.TrimRight(strings.TrimSpace(s), "\x00")
}
