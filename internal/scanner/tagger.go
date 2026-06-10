// Package scanner walks a user's media root, extracts audio metadata and keeps
// the MediaFile table in sync. Tags come from dhowden/tag; duration, bitrate
// and sample rate come from ffprobe (more reliable across containers).
package scanner

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dhowden/tag"
)

// audioExts is the set of recognised audio file extensions.
var audioExts = map[string]bool{
	".mp3": true, ".flac": true, ".ogg": true, ".opus": true,
	".m4a": true, ".aac": true, ".wav": true, ".wma": true,
	".aiff": true, ".aif": true, ".alac": true,
}

// IsAudio reports whether a path has a recognised audio extension.
func IsAudio(path string) bool {
	return audioExts[strings.ToLower(filepath.Ext(path))]
}

// TagInfo is the metadata extracted from one audio file.
type TagInfo struct {
	Title       string
	Album       string
	AlbumArtist string
	Artists     string
	Genres      string
	TrackNo     int
	DiscNo      int
	Year        int

	Duration   float64
	Bitrate    int
	SampleRate int
	Container  string
}

// ParseTags extracts metadata from an audio file at absPath.
func ParseTags(absPath string) (*TagInfo, error) {
	info := &TagInfo{
		Container: strings.TrimPrefix(strings.ToLower(filepath.Ext(absPath)), "."),
	}

	if f, err := os.Open(absPath); err == nil {
		defer f.Close()
		if m, err := tag.ReadFrom(f); err == nil {
			info.Title = m.Title()
			info.Album = m.Album()
			info.AlbumArtist = m.AlbumArtist()
			info.Artists = m.Artist()
			info.Genres = m.Genre()
			info.Year = m.Year()
			info.TrackNo, _ = m.Track()
			info.DiscNo, _ = m.Disc()
		}
	}

	// Fall back to filename for a missing title.
	if info.Title == "" {
		info.Title = strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	}
	if info.AlbumArtist == "" {
		info.AlbumArtist = info.Artists
	}

	probeWithFFprobe(absPath, info)
	return info, nil
}

// ffprobe JSON subset.
type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		SampleRate string `json:"sample_rate"`
		BitRate    string `json:"bit_rate"`
	} `json:"streams"`
}

// probeWithFFprobe fills duration/bitrate/sampleRate using ffprobe if present.
// Absence of ffprobe is non-fatal — fields stay zero.
func probeWithFFprobe(absPath string, info *TagInfo) {
	out, err := exec.Command(
		"ffprobe", "-v", "quiet", "-print_format", "json",
		"-show_format", "-show_streams", absPath,
	).Output()
	if err != nil {
		return
	}

	var p ffprobeOutput
	if json.Unmarshal(out, &p) != nil {
		return
	}

	if d, err := strconv.ParseFloat(p.Format.Duration, 64); err == nil {
		info.Duration = d
	}
	if b, err := strconv.Atoi(p.Format.BitRate); err == nil {
		info.Bitrate = b
	}
	for _, st := range p.Streams {
		if st.CodecType != "audio" {
			continue
		}
		if sr, err := strconv.Atoi(st.SampleRate); err == nil {
			info.SampleRate = sr
		}
		if info.Bitrate == 0 {
			if b, err := strconv.Atoi(st.BitRate); err == nil {
				info.Bitrate = b
			}
		}
		break
	}
}

// hash returns a short stable hex hash of the given parts (lowercased, joined).
func hash(parts ...string) string {
	h := sha1.Sum([]byte(strings.ToLower(strings.Join(parts, "\x00"))))
	return hex.EncodeToString(h[:])[:16]
}

// AlbumHash identifies an album by album-artist + album title.
func AlbumHash(albumArtist, album string) string { return hash(albumArtist, album) }

// ArtistHash identifies an artist by name.
func ArtistHash(artist string) string { return hash(artist) }

// TrackHash identifies a track (for dedup across qualities/paths).
func TrackHash(artists, album, title string) string { return hash(artists, album, title) }
