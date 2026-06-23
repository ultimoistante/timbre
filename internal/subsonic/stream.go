package subsonic

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/stream"
)

// formatToContainer maps a Subsonic stream `format` to an ffmpeg container.
var formatToContainer = map[string]string{
	"mp3":    "mp3",
	"aac":    "aac",
	"opus":   "opus",
	"flac":   "flac",
	"ogg":    "ogg",
	"vorbis": "ogg",
}

// stream serves a track, transcoding when maxBitRate/format request it.
func (h *Handlers) stream(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "id"))
	if err != nil || p.Kind != KindTrack {
		return WriteError(c, ErrNotFound, "Song not found.")
	}
	mf, ok := h.trackByID(u.ID, p.UintID)
	if !ok {
		return WriteError(c, ErrNotFound, "Song not found.")
	}
	abs, err := h.store.Resolve(u.ID, mf.RelPath)
	if err != nil {
		return WriteError(c, ErrNotFound, "Song not found.")
	}

	format := param(c, "format")
	maxBitRate := atoiDefault(param(c, "maxBitRate"), 0)
	container, transcode := formatToContainer[format]

	// Serve original bytes unless an explicit transcodable format is requested.
	if format == "" || format == "raw" || !transcode {
		return stream.ServeOriginal(c.Response().Writer, c.Request(), abs)
	}

	quality := capBitrate(maxBitRate, mf.Bitrate)
	return stream.Transcode(c.Response().Writer, c.Request(), abs, quality, container)
}

// download always serves the original file unmodified.
func (h *Handlers) download(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "id"))
	if err != nil || p.Kind != KindTrack {
		return WriteError(c, ErrNotFound, "Song not found.")
	}
	mf, ok := h.trackByID(u.ID, p.UintID)
	if !ok {
		return WriteError(c, ErrNotFound, "Song not found.")
	}
	abs, err := h.store.Resolve(u.ID, mf.RelPath)
	if err != nil {
		return WriteError(c, ErrNotFound, "Song not found.")
	}
	return stream.ServeOriginal(c.Response().Writer, c.Request(), abs)
}

// getCoverArt serves an album's cover art bytes. The id is a co-/al- hash.
func (h *Handlers) getCoverArt(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "id"))
	if err != nil || (p.Kind != KindCover && p.Kind != KindAlbum) {
		return c.NoContent(http.StatusNotFound)
	}
	data, mime, err := ExtractAlbumArt(h.db, h.store, h.dataDir, u.ID, p.Hash)
	if err != nil {
		if errors.Is(err, ErrNoArt) {
			return c.NoContent(http.StatusNotFound)
		}
		return err
	}
	c.Response().Header().Set("Cache-Control", "public, immutable, max-age=31536000")
	return c.Blob(http.StatusOK, mime, data)
}

// capBitrate converts a requested kbps cap to an ffmpeg bitrate string, never
// exceeding the source bitrate (no upsampling). maxBitRate 0 means "no cap".
func capBitrate(maxKbps, sourceBitsPerSec int) string {
	sourceKbps := sourceBitsPerSec / 1000
	target := maxKbps
	if target == 0 || (sourceKbps > 0 && target > sourceKbps) {
		target = sourceKbps
	}
	if target <= 0 {
		target = 192 // sane fallback when source bitrate is unknown
	}
	return strconv.Itoa(target) + "k"
}
