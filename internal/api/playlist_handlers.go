package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
)

// PlaylistSummary is the list-view shape returned by GET /playlists.
type PlaylistSummary struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Pinned      bool     `json:"pinned"`
	TrackCount  int      `json:"trackCount"`
	AlbumHashes []string `json:"albumHashes"` // up to 4, for mosaic art
}

func (s *Server) handleListPlaylists(c echo.Context) error {
	u := auth.CurrentUser(c)
	var playlists []models.Playlist
	if err := s.db.Where("user_id = ?", u.ID).Order("pinned DESC, name").Find(&playlists).Error; err != nil {
		return err
	}

	summaries := make([]PlaylistSummary, 0, len(playlists))
	for _, pl := range playlists {
		var count int64
		s.db.Model(&models.PlaylistTrack{}).Where("playlist_id = ?", pl.ID).Count(&count)

		var rows []struct{ AlbumHash string }
		s.db.Model(&models.MediaFile{}).
			Joins("JOIN playlist_tracks ON playlist_tracks.track_id = media_files.id").
			Where("playlist_tracks.playlist_id = ?", pl.ID).
			Select("DISTINCT media_files.album_hash").
			Limit(4).
			Scan(&rows)

		hashes := make([]string, 0, len(rows))
		for _, r := range rows {
			if r.AlbumHash != "" {
				hashes = append(hashes, r.AlbumHash)
			}
		}

		summaries = append(summaries, PlaylistSummary{
			ID:          pl.ID,
			Name:        pl.Name,
			Description: pl.Description,
			Pinned:      pl.Pinned,
			TrackCount:  int(count),
			AlbumHashes: hashes,
		})
	}
	return c.JSON(http.StatusOK, summaries)
}

func (s *Server) handleCreatePlaylist(c echo.Context) error {
	u := auth.CurrentUser(c)
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&body); err != nil || body.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name required"})
	}
	pl := models.Playlist{UserID: u.ID, Name: body.Name, Description: body.Description}
	if err := s.db.Create(&pl).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, PlaylistSummary{
		ID: pl.ID, Name: pl.Name, Description: pl.Description,
		Pinned: pl.Pinned, TrackCount: 0, AlbumHashes: []string{},
	})
}

func (s *Server) handleGetPlaylist(c echo.Context) error {
	u := auth.CurrentUser(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var pl models.Playlist
	if err := s.db.Where("id = ? AND user_id = ?", id, u.ID).First(&pl).Error; err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	var pts []models.PlaylistTrack
	s.db.Where("playlist_id = ?", pl.ID).Order("position").Find(&pts)

	trackIDs := make([]uint, len(pts))
	for i, pt := range pts {
		trackIDs[i] = pt.TrackID
	}

	var tracks []models.MediaFile
	if len(trackIDs) > 0 {
		s.db.Where("id IN ? AND user_id = ?", trackIDs, u.ID).Find(&tracks)
		// preserve playlist order
		byID := make(map[uint]models.MediaFile, len(tracks))
		for _, t := range tracks {
			byID[t.ID] = t
		}
		ordered := make([]models.MediaFile, 0, len(trackIDs))
		for _, id := range trackIDs {
			if t, ok := byID[id]; ok {
				ordered = append(ordered, t)
			}
		}
		tracks = ordered
	}

	return c.JSON(http.StatusOK, map[string]any{
		"id":          pl.ID,
		"name":        pl.Name,
		"description": pl.Description,
		"pinned":      pl.Pinned,
		"tracks":      tracks,
	})
}

func (s *Server) handleUpdatePlaylist(c echo.Context) error {
	u := auth.CurrentUser(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var pl models.Playlist
	if err := s.db.Where("id = ? AND user_id = ?", id, u.ID).First(&pl).Error; err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Pinned      *bool   `json:"pinned"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if body.Name != nil {
		pl.Name = *body.Name
	}
	if body.Description != nil {
		pl.Description = *body.Description
	}
	if body.Pinned != nil {
		pl.Pinned = *body.Pinned
	}
	s.db.Save(&pl)
	return c.JSON(http.StatusOK, map[string]any{"id": pl.ID, "name": pl.Name, "description": pl.Description, "pinned": pl.Pinned})
}

func (s *Server) handleDeletePlaylist(c echo.Context) error {
	u := auth.CurrentUser(c)
	id, _ := strconv.Atoi(c.Param("id"))
	s.db.Where("id = ? AND user_id = ?", id, u.ID).Delete(&models.Playlist{})
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleAddPlaylistTracks(c echo.Context) error {
	u := auth.CurrentUser(c)
	plID, _ := strconv.Atoi(c.Param("id"))

	var pl models.Playlist
	if err := s.db.Where("id = ? AND user_id = ?", plID, u.ID).First(&pl).Error; err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	var body struct {
		TrackIDs []uint `json:"trackIds"`
	}
	if err := c.Bind(&body); err != nil || len(body.TrackIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "trackIds required"})
	}

	var maxPos int
	s.db.Model(&models.PlaylistTrack{}).Where("playlist_id = ?", plID).Select("COALESCE(MAX(position),0)").Scan(&maxPos)

	for i, tid := range body.TrackIDs {
		s.db.Create(&models.PlaylistTrack{PlaylistID: uint(plID), TrackID: tid, Position: maxPos + i + 1})
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleRemovePlaylistTrack(c echo.Context) error {
	u := auth.CurrentUser(c)
	plID, _ := strconv.Atoi(c.Param("id"))
	ptID, _ := strconv.Atoi(c.Param("ptId"))

	var pl models.Playlist
	if err := s.db.Where("id = ? AND user_id = ?", plID, u.ID).First(&pl).Error; err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	s.db.Where("id = ? AND playlist_id = ?", ptID, plID).Delete(&models.PlaylistTrack{})
	return c.NoContent(http.StatusNoContent)
}
