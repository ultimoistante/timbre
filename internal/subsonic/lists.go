package subsonic

import (
	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/models"
)

// albumListOrder maps a getAlbumList2 `type` to an ORDER BY clause on the
// derived-album query.
var albumListOrder = map[string]string{
	"newest":               "MAX(created_at) DESC",
	"recent":               "MAX(last_played) DESC",
	"frequent":             "SUM(play_count) DESC",
	"alphabeticalByName":   "album ASC",
	"alphabeticalByArtist": "album_artist ASC",
	"random":               "RANDOM()",
	"byYear":               "MAX(year) ASC",
	"byGenre":              "album ASC",
}

// albumListQuery resolves the order/filter from the request params shared by
// getAlbumList and getAlbumList2.
func (h *Handlers) albumListQuery(c echo.Context, userID uint) []albumAgg {
	typ := param(c, "type")
	if typ == "" {
		typ = "alphabeticalByName"
	}
	order, ok := albumListOrder[typ]
	if !ok {
		order = "album ASC"
	}
	size := atoiDefault(param(c, "size"), 10)
	if size > 500 {
		size = 500
	}
	offset := atoiDefault(param(c, "offset"), 0)

	where, args := "", ""
	switch typ {
	case "byGenre":
		where, args = "genres LIKE ?", "%"+param(c, "genre")+"%"
	case "byYear":
		from := atoiDefault(param(c, "fromYear"), 0)
		to := atoiDefault(param(c, "toYear"), 0)
		if from > to && to > 0 {
			from, to = to, from
			order = "MAX(year) DESC"
		}
	}
	return h.queryAlbums(userID, where, args, order, size, offset)
}

// getAlbumList2 returns albums (ID3) by the requested list type.
func (h *Handlers) getAlbumList2(c echo.Context) error {
	u := CurrentUser(c)
	albums := h.albumListQuery(c, u.ID)
	out := make([]AlbumID3, len(albums))
	for i, a := range albums {
		out[i] = albumID3FromAgg(a)
	}
	return Write(c, func(r *Response) {
		r.AlbumList2 = &AlbumList2{Album: out}
	})
}

// getAlbumList returns albums (legacy directory style) by list type.
func (h *Handlers) getAlbumList(c echo.Context) error {
	u := CurrentUser(c)
	albums := h.albumListQuery(c, u.ID)
	out := make([]Child, len(albums))
	for i, a := range albums {
		out[i] = dirChildFromAlbum(a)
	}
	return Write(c, func(r *Response) {
		r.AlbumList = &AlbumList{Album: out}
	})
}

// getRandomSongs returns a random selection of songs, optionally filtered.
func (h *Handlers) getRandomSongs(c echo.Context) error {
	u := CurrentUser(c)
	size := atoiDefault(param(c, "size"), 10)
	if size > 500 {
		size = 500
	}

	q := h.db.Where("user_id = ?", u.ID)
	if g := param(c, "genre"); g != "" {
		q = q.Where("genres LIKE ?", "%"+g+"%")
	}
	if from := atoiDefault(param(c, "fromYear"), 0); from > 0 {
		q = q.Where("year >= ?", from)
	}
	if to := atoiDefault(param(c, "toYear"), 0); to > 0 {
		q = q.Where("year <= ?", to)
	}

	var tracks []models.MediaFile
	q.Order("RANDOM()").Limit(size).Find(&tracks)
	songs := make([]Child, len(tracks))
	for i, t := range tracks {
		songs[i] = songFromMediaFile(t)
	}
	return Write(c, func(r *Response) {
		r.RandomSongs = &Songs{Song: songs}
	})
}

// getNowPlaying returns an empty list (timbre tracks no cross-client playback).
func (h *Handlers) getNowPlaying(c echo.Context) error {
	return Write(c, func(r *Response) {
		r.NowPlaying = &NowPlaying{}
	})
}
