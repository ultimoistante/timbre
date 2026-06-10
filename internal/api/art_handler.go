package api

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/dhowden/tag"
	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
)

var validHash = regexp.MustCompile(`^[0-9a-f]{16}$`)

// handleAlbumArt extracts the cover art for an album from one of its audio
// files. The result is cached on disk so subsequent requests are served from
// the cache. Returns 404 if no picture is embedded.
func (s *Server) handleAlbumArt(c echo.Context) error {
	albumHash := c.Param("hash")
	if !validHash.MatchString(albumHash) {
		return c.NoContent(http.StatusNotFound)
	}
	u := auth.CurrentUser(c)

	// Art cache dir: <DataDir>/art/
	cacheDir := filepath.Join(s.cfg.DataDir, "art")
	cachePath := filepath.Join(cacheDir, albumHash)

	const cacheControl = "public, immutable, max-age=31536000"

	// Serve from cache if present.
	if data, err := os.ReadFile(cachePath); err == nil {
		mime := http.DetectContentType(data)
		c.Response().Header().Set("Cache-Control", cacheControl)
		return c.Blob(http.StatusOK, mime, data)
	}

	// Find any track belonging to this album+user.
	var track models.MediaFile
	if err := s.db.Where("user_id = ? AND album_hash = ?", u.ID, albumHash).
		First(&track).Error; err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	// Resolve absolute path.
	absPath, err := s.store.Resolve(u.ID, track.RelPath)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	// Read tags and extract picture.
	f, err := os.Open(absPath)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	pic := m.Picture()
	if pic == nil || len(pic.Data) == 0 {
		return c.NoContent(http.StatusNotFound)
	}

	// Cache to disk (best-effort — ignore write errors).
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		_ = os.WriteFile(cachePath, pic.Data, 0o644)
	}

	mime := pic.MIMEType
	if mime == "" || mime == "application/octet-stream" {
		mime = http.DetectContentType(pic.Data)
	}
	c.Response().Header().Set("Cache-Control", cacheControl)
	return c.Blob(http.StatusOK, mime, pic.Data)
}
