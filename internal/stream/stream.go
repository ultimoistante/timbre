// Package stream serves audio files with Range support and optionally
// transcodes them on-the-fly via ffmpeg. Original files use http.ServeContent
// (Range/seek free). Transcode pipes ffmpeg stdout directly to the response —
// no temp files, no polling.
package stream

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// mimeTypes maps lowercase extension to MIME type.
var mimeTypes = map[string]string{
	"mp3":  "audio/mpeg",
	"flac": "audio/flac",
	"ogg":  "audio/ogg",
	"opus": "audio/ogg; codecs=opus",
	"m4a":  "audio/mp4",
	"aac":  "audio/aac",
	"wav":  "audio/wav",
	"webm": "audio/webm",
	"wma":  "audio/x-ms-wma",
	"aiff": "audio/aiff",
	"aif":  "audio/aiff",
}

// MimeType returns the audio MIME type for the given file extension (no dot).
func MimeType(ext string) string {
	if m, ok := mimeTypes[strings.ToLower(ext)]; ok {
		return m
	}
	return "application/octet-stream"
}

// ServeOriginal streams a file unmodified using http.ServeContent, which
// handles Range requests, ETags and conditional GET automatically.
func ServeOriginal(w http.ResponseWriter, r *http.Request, absPath string) error {
	f, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(absPath)), ".")
	w.Header().Set("Content-Type", MimeType(ext))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "no-cache")

	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
	return nil
}

// containerArgs returns the ffmpeg codec+format flags for the requested output
// container. The "-f" flag is mandatory for piped output (pipe:1 has no
// extension for ffmpeg to infer the container).
var containerArgs = map[string][]string{
	"mp3":  {"-c:a", "libmp3lame", "-f", "mp3"},
	"aac":  {"-c:a", "aac", "-f", "adts"},
	"ogg":  {"-c:a", "libvorbis", "-f", "ogg"},
	"opus": {"-c:a", "libopus", "-f", "ogg"},
	"flac": {"-c:a", "flac", "-f", "flac"},
}

// Transcode pipes ffmpeg output for absPath directly into w. quality is the
// target bitrate string (e.g. "128k"); container selects the codec/format.
// The response is not seekable (no Content-Length), but streaming starts
// immediately.
func Transcode(w http.ResponseWriter, r *http.Request, absPath, quality, container string) error {
	args, ok := containerArgs[container]
	if !ok {
		return fmt.Errorf("unsupported container: %q", container)
	}

	ext := container
	if container == "opus" {
		ext = "ogg"
	}
	w.Header().Set("Content-Type", MimeType(ext))
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Transcoded-Bitrate", quality)
	w.Header().Set("Transfer-Encoding", "chunked")

	cmd := exec.CommandContext(r.Context(), "ffmpeg",
		append([]string{
			"-hide_banner", "-loglevel", "error",
			"-i", absPath,
			"-vn",                    // drop video/cover streams
			"-map_metadata", "0",     // preserve audio metadata
			"-b:a", quality,
		}, append(args, "pipe:1")...)...,
	)
	cmd.Stdout = w
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}
	_ = cmd.Wait() // client disconnect cancels via context
	return nil
}

// ModTimeFromPath returns the modification time of absPath, falling back to
// now on error — used as ETag seed for http.ServeContent.
func ModTimeFromPath(absPath string) time.Time {
	fi, err := os.Stat(absPath)
	if err != nil {
		return time.Now()
	}
	return fi.ModTime()
}
