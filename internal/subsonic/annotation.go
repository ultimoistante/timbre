package subsonic

import (
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/ultimoistante/timbre/internal/models"
)

// scrobble registers a play: it bumps PlayCount and LastPlayed for the track.
// The submission=false (now-playing notification) case is accepted as a no-op.
func (h *Handlers) scrobble(c echo.Context) error {
	u := CurrentUser(c)
	submission := param(c, "submission")
	if submission == "false" {
		return Write(c, nil)
	}
	for _, raw := range paramValues(c, "id") {
		p, err := ParseID(raw)
		if err != nil || p.Kind != KindTrack {
			continue
		}
		h.db.Model(&models.MediaFile{}).
			Where("user_id = ? AND id = ?", u.ID, p.UintID).
			Updates(map[string]any{
				"play_count":  gorm.Expr("play_count + 1"),
				"last_played": time.Now().Unix(),
			})
	}
	return Write(c, nil)
}

// star adds the given items to the user's favorites.
func (h *Handlers) star(c echo.Context) error {
	u := CurrentUser(c)
	for _, raw := range h.starItems(c) {
		st := models.Star{UserID: u.ID, ItemID: raw.itemID, ItemType: raw.itemType}
		// Upsert: ignore duplicates (unique index on user_id+item_id).
		h.db.Where("user_id = ? AND item_id = ?", u.ID, raw.itemID).
			FirstOrCreate(&st)
	}
	return Write(c, nil)
}

// unstar removes the given items from the user's favorites.
func (h *Handlers) unstar(c echo.Context) error {
	u := CurrentUser(c)
	for _, raw := range h.starItems(c) {
		h.db.Where("user_id = ? AND item_id = ?", u.ID, raw.itemID).
			Delete(&models.Star{})
	}
	return Write(c, nil)
}

// setRating is accepted but not stored (timbre has no per-item ratings).
func (h *Handlers) setRating(c echo.Context) error {
	return Write(c, nil)
}

// getStarred2 returns starred artists, albums and songs (ID3).
func (h *Handlers) getStarred2(c echo.Context) error {
	u := CurrentUser(c)
	artists, albums, songs := h.collectStarred(u.ID)
	artOut := make([]ArtistID3, len(artists))
	for i, a := range artists {
		artOut[i] = artistID3FromAgg(a)
	}
	albOut := make([]AlbumID3, len(albums))
	for i, a := range albums {
		albOut[i] = albumID3FromAgg(a)
	}
	songOut := make([]Child, len(songs))
	for i, s := range songs {
		songOut[i] = songFromMediaFile(s)
	}
	return Write(c, func(r *Response) {
		r.Starred2 = &Starred2{Artist: artOut, Album: albOut, Song: songOut}
	})
}

// getStarred is the legacy variant (albums as directory entries).
func (h *Handlers) getStarred(c echo.Context) error {
	u := CurrentUser(c)
	artists, albums, songs := h.collectStarred(u.ID)
	artOut := make([]ArtistID3, len(artists))
	for i, a := range artists {
		artOut[i] = artistID3FromAgg(a)
	}
	albOut := make([]Child, len(albums))
	for i, a := range albums {
		albOut[i] = dirChildFromAlbum(a)
	}
	songOut := make([]Child, len(songs))
	for i, s := range songs {
		songOut[i] = songFromMediaFile(s)
	}
	return Write(c, func(r *Response) {
		r.Starred = &Starred{Artist: artOut, Album: albOut, Song: songOut}
	})
}

type starItem struct {
	itemID   string
	itemType string
}

// starItems parses the id/albumId/artistId params of star/unstar into typed ids.
func (h *Handlers) starItems(c echo.Context) []starItem {
	var items []starItem
	add := func(raw string) {
		p, err := ParseID(raw)
		if err != nil {
			return
		}
		switch p.Kind {
		case KindTrack:
			items = append(items, starItem{TrackID(p.UintID), "track"})
		case KindAlbum:
			items = append(items, starItem{AlbumID(p.Hash), "album"})
		case KindArtist:
			items = append(items, starItem{ArtistID(p.Hash), "artist"})
		}
	}
	for _, raw := range paramValues(c, "id") {
		add(raw)
	}
	for _, raw := range paramValues(c, "albumId") {
		add(raw)
	}
	for _, raw := range paramValues(c, "artistId") {
		add(raw)
	}
	return items
}

// collectStarred resolves the user's stars back into library entities.
func (h *Handlers) collectStarred(userID uint) ([]artistAgg, []albumAgg, []models.MediaFile) {
	var stars []models.Star
	h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&stars)

	var artists []artistAgg
	var albums []albumAgg
	var songs []models.MediaFile
	for _, st := range stars {
		p, err := ParseID(st.ItemID)
		if err != nil {
			continue
		}
		switch p.Kind {
		case KindArtist:
			albs := h.queryAlbums(userID, "artist_hash = ?", p.Hash, "album", 1, 0)
			name := ""
			if len(albs) > 0 {
				name = albs[0].AlbumArtist
			}
			var cnt int64
			h.db.Model(&models.MediaFile{}).
				Where("user_id = ? AND artist_hash = ?", userID, p.Hash).
				Distinct("album_hash").Count(&cnt)
			artists = append(artists, artistAgg{ArtistHash: p.Hash, Name: name, AlbumCount: int(cnt)})
		case KindAlbum:
			if a, ok := h.albumByHash(userID, p.Hash); ok {
				albums = append(albums, a)
			}
		case KindTrack:
			if mf, ok := h.trackByID(userID, p.UintID); ok {
				songs = append(songs, mf)
			}
		}
	}
	return artists, albums, songs
}
