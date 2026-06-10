package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
)

// paginate reads limit/offset query params with sane bounds.
func paginate(c echo.Context) (limit, offset int) {
	limit = 100
	if v, err := strconv.Atoi(c.QueryParam("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > 500 {
		limit = 500
	}
	if v, err := strconv.Atoi(c.QueryParam("offset")); err == nil && v > 0 {
		offset = v
	}
	return
}

// handleTracks lists the current user's tracks, optionally filtered by album
// hash, artist hash, or a free-text query.
func (s *Server) handleTracks(c echo.Context) error {
	u := auth.CurrentUser(c)
	limit, offset := paginate(c)

	q := s.db.Model(&models.MediaFile{}).Where("user_id = ?", u.ID)
	if h := c.QueryParam("album"); h != "" {
		q = q.Where("album_hash = ?", h)
	}
	if h := c.QueryParam("artist"); h != "" {
		q = q.Where("artist_hash = ?", h)
	}
	if term := c.QueryParam("q"); term != "" {
		like := "%" + term + "%"
		q = q.Where("title LIKE ? OR album LIKE ? OR artists LIKE ?", like, like, like)
	}

	var tracks []models.MediaFile
	if err := q.Order("album_artist, album, disc_no, track_no").
		Limit(limit).Offset(offset).Find(&tracks).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, tracks)
}

// AlbumAgg is a derived album row.
type AlbumAgg struct {
	AlbumHash   string  `json:"albumHash"`
	Album       string  `json:"album"`
	AlbumArtist string  `json:"albumArtist"`
	ArtistHash  string  `json:"artistHash"`
	Year        int     `json:"year"`
	TrackCount  int     `json:"trackCount"`
	Duration    float64 `json:"duration"`
}

// handleAlbums lists albums derived from the user's tracks.
func (s *Server) handleAlbums(c echo.Context) error {
	u := auth.CurrentUser(c)
	limit, offset := paginate(c)

	var albums []AlbumAgg
	if err := s.db.Model(&models.MediaFile{}).
		Select(`album_hash, album, album_artist, artist_hash,
			MAX(year) as year, COUNT(*) as track_count, SUM(duration) as duration`).
		Where("user_id = ?", u.ID).
		Group("album_hash, album, album_artist, artist_hash").
		Order("album_artist, album").
		Limit(limit).Offset(offset).
		Scan(&albums).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, albums)
}

// handleAlbumTracks lists the tracks of one album.
func (s *Server) handleAlbumTracks(c echo.Context) error {
	u := auth.CurrentUser(c)
	var tracks []models.MediaFile
	if err := s.db.Where("user_id = ? AND album_hash = ?", u.ID, c.Param("hash")).
		Order("disc_no, track_no").Find(&tracks).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, tracks)
}

// ArtistAgg is a derived artist row.
type ArtistAgg struct {
	ArtistHash string `json:"artistHash"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"`
	TrackCount int    `json:"trackCount"`
}

// handleArtists lists artists derived from the user's tracks.
func (s *Server) handleArtists(c echo.Context) error {
	u := auth.CurrentUser(c)
	limit, offset := paginate(c)

	var artists []ArtistAgg
	if err := s.db.Model(&models.MediaFile{}).
		Select(`artist_hash, album_artist as name,
			COUNT(DISTINCT album_hash) as album_count, COUNT(*) as track_count`).
		Where("user_id = ?", u.ID).
		Group("artist_hash, album_artist").
		Order("album_artist").
		Limit(limit).Offset(offset).
		Scan(&artists).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, artists)
}

// handleSearch does a simple cross-field search over the user's tracks.
func (s *Server) handleSearch(c echo.Context) error {
	u := auth.CurrentUser(c)
	term := c.QueryParam("q")
	if term == "" {
		return c.JSON(http.StatusOK, []models.MediaFile{})
	}
	like := "%" + term + "%"
	limit, _ := paginate(c)

	var tracks []models.MediaFile
	if err := s.db.Where("user_id = ?", u.ID).
		Where("title LIKE ? OR album LIKE ? OR album_artist LIKE ? OR artists LIKE ?",
			like, like, like, like).
		Limit(limit).Find(&tracks).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, tracks)
}

// handleRecentlyAdded returns the N most recently added albums for the current user,
// ordered by the most recent track added to each album.
func (s *Server) handleRecentlyAdded(c echo.Context) error {
	u := auth.CurrentUser(c)

	var albums []AlbumAgg
	if err := s.db.Model(&models.MediaFile{}).
		Select(`album_hash, album, album_artist, artist_hash,
			MAX(year) as year, COUNT(*) as track_count, SUM(duration) as duration`).
		Where("user_id = ?", u.ID).
		Group("album_hash, album, album_artist, artist_hash").
		Order("MAX(created_at) DESC").
		Limit(20).
		Scan(&albums).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, albums)
}
