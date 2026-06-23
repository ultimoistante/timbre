package api

import (
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/subsonic"
)

var validHash = regexp.MustCompile(`^[0-9a-f]{16}$`)

// handleAlbumArt extracts the cover art for an album from one of its audio
// files. The result is cached on disk so subsequent requests are served from
// the cache. Returns 404 if no picture is embedded. The extraction/caching is
// shared with the Subsonic getCoverArt endpoint (subsonic.ExtractAlbumArt).
func (s *Server) handleAlbumArt(c echo.Context) error {
	albumHash := c.Param("hash")
	if !validHash.MatchString(albumHash) {
		return c.NoContent(http.StatusNotFound)
	}
	u := auth.CurrentUser(c)

	data, mime, err := subsonic.ExtractAlbumArt(s.db, s.store, s.cfg.DataDir, u.ID, albumHash)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	c.Response().Header().Set("Cache-Control", "public, immutable, max-age=31536000")
	return c.Blob(http.StatusOK, mime, data)
}
