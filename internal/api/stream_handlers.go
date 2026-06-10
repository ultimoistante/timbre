package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
	"github.com/ultimoistante/timbre/internal/stream"
)

// handleStream serves a track by MediaFile ID. Query params:
//   - quality  : target bitrate for transcode (e.g. "128k", "320k"). Absent or
//     "original" → serve original file with Range/seek support.
//   - container: output format for transcode (mp3/aac/ogg/opus/flac).
//     Defaults to "mp3" when quality is set.
func (s *Server) handleStream(c echo.Context) error {
	u := auth.CurrentUser(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var mf models.MediaFile
	if err := s.db.Where("user_id = ? AND id = ?", u.ID, id).First(&mf).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "track not found")
	}

	// Resolve absolute path for the track.
	absPath, err := s.store.Resolve(u.ID, mf.RelPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "path error")
	}

	quality := c.QueryParam("quality")
	container := c.QueryParam("container")

	// Original stream (default when quality absent/original).
	if quality == "" || quality == "original" {
		if err := stream.ServeOriginal(c.Response().Writer, c.Request(), absPath); err != nil {
			if isGone(err) {
				return echo.NewHTTPError(http.StatusNotFound, "file not found on disk")
			}
			return err
		}
		return nil
	}

	// Transcode: quality like "128k"; container defaults to mp3.
	if container == "" {
		container = "mp3"
	}

	// Cap bitrate to file's original bitrate to avoid upsample bloat.
	quality = capBitrate(quality, mf.Bitrate)

	if err := stream.Transcode(c.Response().Writer, c.Request(), absPath, quality, container); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

// capBitrate returns the lower of requested and the file's original bitrate.
// original is in bits/sec from GORM; quality is like "128k".
func capBitrate(quality string, originalBps int) string {
	if originalBps <= 0 {
		return quality
	}
	req := parseKbps(quality)
	orig := originalBps / 1000
	if req <= orig {
		return quality
	}
	return strconv.Itoa(orig) + "k"
}

func parseKbps(q string) int {
	s := q
	if len(s) > 0 && (s[len(s)-1] == 'k' || s[len(s)-1] == 'K') {
		s = s[:len(s)-1]
	}
	n, _ := strconv.Atoi(s)
	return n
}

func isGone(err error) bool {
	// os.PathError wraps os.ErrNotExist.
	return err != nil && (err.Error() == "file not found" ||
		containsStr(err.Error(), "no such file"))
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsAny(s, sub))
}

func containsAny(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
