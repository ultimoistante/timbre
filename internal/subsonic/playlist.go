package subsonic

import (
	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/models"
)

// playlistSummary builds a PlaylistDTO with computed song count and duration.
func (h *Handlers) playlistSummary(pl models.Playlist, owner string) PlaylistDTO {
	var count int64
	h.db.Model(&models.PlaylistTrack{}).Where("playlist_id = ?", pl.ID).Count(&count)

	var dur float64
	h.db.Model(&models.MediaFile{}).
		Joins("JOIN playlist_tracks ON playlist_tracks.track_id = media_files.id").
		Where("playlist_tracks.playlist_id = ?", pl.ID).
		Select("COALESCE(SUM(media_files.duration), 0)").Scan(&dur)

	return PlaylistDTO{
		ID:        PlaylistID(pl.ID),
		Name:      pl.Name,
		Comment:   pl.Description,
		Owner:     owner,
		SongCount: int(count),
		Duration:  int(dur + 0.5),
		Public:    false,
		Created:   timeRFC(pl.CreatedAt),
		Changed:   timeRFC(pl.UpdatedAt),
	}
}

// playlistTracks returns a playlist's tracks in playlist order.
func (h *Handlers) playlistTracks(userID, plID uint) []models.MediaFile {
	var pts []models.PlaylistTrack
	h.db.Where("playlist_id = ?", plID).Order("position").Find(&pts)
	if len(pts) == 0 {
		return nil
	}
	ids := make([]uint, len(pts))
	for i, pt := range pts {
		ids[i] = pt.TrackID
	}
	var tracks []models.MediaFile
	h.db.Where("id IN ? AND user_id = ?", ids, userID).Find(&tracks)
	byID := make(map[uint]models.MediaFile, len(tracks))
	for _, t := range tracks {
		byID[t.ID] = t
	}
	ordered := make([]models.MediaFile, 0, len(ids))
	for _, id := range ids {
		if t, ok := byID[id]; ok {
			ordered = append(ordered, t)
		}
	}
	return ordered
}

// getPlaylists lists the user's playlists.
func (h *Handlers) getPlaylists(c echo.Context) error {
	u := CurrentUser(c)
	var pls []models.Playlist
	h.db.Where("user_id = ?", u.ID).Order("pinned DESC, name").Find(&pls)

	out := make([]PlaylistDTO, len(pls))
	for i, pl := range pls {
		out[i] = h.playlistSummary(pl, u.Username)
	}
	return Write(c, func(r *Response) {
		r.Playlists = &Playlists{Playlist: out}
	})
}

// getPlaylist returns one playlist with its songs.
func (h *Handlers) getPlaylist(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "id"))
	if err != nil || p.Kind != KindPlaylist {
		return WriteError(c, ErrNotFound, "Playlist not found.")
	}
	var pl models.Playlist
	if err := h.db.Where("id = ? AND user_id = ?", p.UintID, u.ID).First(&pl).Error; err != nil {
		return WriteError(c, ErrNotFound, "Playlist not found.")
	}

	tracks := h.playlistTracks(u.ID, pl.ID)
	entries := make([]Child, len(tracks))
	for i, t := range tracks {
		entries[i] = songFromMediaFile(t)
	}
	return Write(c, func(r *Response) {
		r.Playlist = &PlaylistWithSongs{
			PlaylistDTO: h.playlistSummary(pl, u.Username),
			Entry:       entries,
		}
	})
}

// createPlaylist creates a playlist, optionally seeded with songId values.
func (h *Handlers) createPlaylist(c echo.Context) error {
	u := CurrentUser(c)

	// If a playlistId is supplied, treat as a full replace of that playlist.
	if pid := param(c, "playlistId"); pid != "" {
		p, err := ParseID(pid)
		if err == nil && p.Kind == KindPlaylist {
			h.replacePlaylistTracks(u.ID, p.UintID, h.songIDsParam(c, "songId"))
			return h.respondPlaylist(c, u, p.UintID)
		}
	}

	name := param(c, "name")
	if name == "" {
		return WriteError(c, ErrMissingPar, "Required parameter 'name' is missing.")
	}
	pl := models.Playlist{UserID: u.ID, Name: name}
	if err := h.db.Create(&pl).Error; err != nil {
		return WriteError(c, ErrGeneric, "Could not create playlist.")
	}
	h.appendPlaylistTracks(pl.ID, h.songIDsParam(c, "songId"))
	return h.respondPlaylist(c, u, pl.ID)
}

// updatePlaylist renames a playlist and/or adds/removes songs.
func (h *Handlers) updatePlaylist(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "playlistId"))
	if err != nil || p.Kind != KindPlaylist {
		return WriteError(c, ErrNotFound, "Playlist not found.")
	}
	var pl models.Playlist
	if err := h.db.Where("id = ? AND user_id = ?", p.UintID, u.ID).First(&pl).Error; err != nil {
		return WriteError(c, ErrNotFound, "Playlist not found.")
	}

	if name := param(c, "name"); name != "" {
		pl.Name = name
	}
	if comment := param(c, "comment"); comment != "" {
		pl.Description = comment
	}
	h.db.Save(&pl)

	// Add new songs.
	h.appendPlaylistTracks(pl.ID, h.songIDsParam(c, "songIdToAdd"))

	// Remove by index (positions in the current ordering, descending so earlier
	// removals don't shift later indices).
	if removeRaw := paramValues(c, "songIndexToRemove"); len(removeRaw) > 0 {
		h.removePlaylistIndices(pl.ID, removeRaw)
	}

	return Write(c, nil)
}

// deletePlaylist removes a playlist.
func (h *Handlers) deletePlaylist(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "id"))
	if err != nil || p.Kind != KindPlaylist {
		return WriteError(c, ErrNotFound, "Playlist not found.")
	}
	h.db.Where("id = ? AND user_id = ?", p.UintID, u.ID).Delete(&models.Playlist{})
	h.db.Where("playlist_id = ?", p.UintID).Delete(&models.PlaylistTrack{})
	return Write(c, nil)
}

func (h *Handlers) respondPlaylist(c echo.Context, u *models.User, plID uint) error {
	var pl models.Playlist
	if err := h.db.Where("id = ? AND user_id = ?", plID, u.ID).First(&pl).Error; err != nil {
		return WriteError(c, ErrNotFound, "Playlist not found.")
	}
	tracks := h.playlistTracks(u.ID, pl.ID)
	entries := make([]Child, len(tracks))
	for i, t := range tracks {
		entries[i] = songFromMediaFile(t)
	}
	return Write(c, func(r *Response) {
		r.Playlist = &PlaylistWithSongs{
			PlaylistDTO: h.playlistSummary(pl, u.Username),
			Entry:       entries,
		}
	})
}

// appendPlaylistTracks adds tracks to the end of a playlist.
func (h *Handlers) appendPlaylistTracks(plID uint, trackIDs []uint) {
	if len(trackIDs) == 0 {
		return
	}
	var maxPos int
	h.db.Model(&models.PlaylistTrack{}).Where("playlist_id = ?", plID).
		Select("COALESCE(MAX(position),0)").Scan(&maxPos)
	for i, tid := range trackIDs {
		h.db.Create(&models.PlaylistTrack{PlaylistID: plID, TrackID: tid, Position: maxPos + i + 1})
	}
}

// replacePlaylistTracks clears a playlist and sets it to the given tracks.
func (h *Handlers) replacePlaylistTracks(userID, plID uint, trackIDs []uint) {
	var pl models.Playlist
	if err := h.db.Where("id = ? AND user_id = ?", plID, userID).First(&pl).Error; err != nil {
		return
	}
	h.db.Where("playlist_id = ?", plID).Delete(&models.PlaylistTrack{})
	h.appendPlaylistTracks(plID, trackIDs)
}

// removePlaylistIndices deletes playlist entries at the given 0-based positions
// in the current ordering.
func (h *Handlers) removePlaylistIndices(plID uint, indices []string) {
	var pts []models.PlaylistTrack
	h.db.Where("playlist_id = ?", plID).Order("position").Find(&pts)
	for _, raw := range indices {
		idx := atoiDefault(raw, -1)
		if idx >= 0 && idx < len(pts) {
			h.db.Delete(&models.PlaylistTrack{}, pts[idx].ID)
		}
	}
}

// songIDsParam parses repeated songId params (tr-N) into track ids.
func (h *Handlers) songIDsParam(c echo.Context, name string) []uint {
	var ids []uint
	for _, raw := range paramValues(c, name) {
		if p, err := ParseID(raw); err == nil && p.Kind == KindTrack {
			ids = append(ids, p.UintID)
		}
	}
	return ids
}

// paramValues returns all values of a repeated query/form param.
func paramValues(c echo.Context, name string) []string {
	if vs := c.QueryParams()[name]; len(vs) > 0 {
		return vs
	}
	if form, err := c.FormParams(); err == nil {
		return form[name]
	}
	return nil
}
