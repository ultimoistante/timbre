package subsonic

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/models"
)

// search3 searches artists, albums and songs (ID3). An empty query matches all
// (clients use this to browse the whole library).
func (h *Handlers) search3(c echo.Context) error {
	u := CurrentUser(c)
	artists, albums, songs := h.runSearch(c, u.ID)

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
		r.SearchResult3 = &SearchResult3{Artist: artOut, Album: albOut, Song: songOut}
	})
}

// search2 is the legacy variant (albums are directory-style entries).
func (h *Handlers) search2(c echo.Context) error {
	u := CurrentUser(c)
	artists, albums, songs := h.runSearch(c, u.ID)

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
		r.SearchResult2 = &SearchResult2{Artist: artOut, Album: albOut, Song: songOut}
	})
}

// runSearch resolves the shared search params and returns matched artists,
// albums and songs.
func (h *Handlers) runSearch(c echo.Context, userID uint) ([]artistAgg, []albumAgg, []models.MediaFile) {
	query := strings.Trim(param(c, "query"), `"`)
	like := "%" + query + "%"
	matchAll := query == ""

	artistCount := atoiDefault(param(c, "artistCount"), 20)
	albumCount := atoiDefault(param(c, "albumCount"), 20)
	songCount := atoiDefault(param(c, "songCount"), 20)
	artistOffset := atoiDefault(param(c, "artistOffset"), 0)
	albumOffset := atoiDefault(param(c, "albumOffset"), 0)
	songOffset := atoiDefault(param(c, "songOffset"), 0)

	// Artists.
	var artists []artistAgg
	aq := h.db.Model(&models.MediaFile{}).
		Select(`artist_hash, album_artist as name,
			COUNT(DISTINCT album_hash) as album_count, COUNT(*) as track_count`).
		Where("user_id = ? AND album_artist <> ''", userID)
	if !matchAll {
		aq = aq.Where("album_artist LIKE ?", like)
	}
	aq.Group("artist_hash, album_artist").Order("album_artist").
		Limit(artistCount).Offset(artistOffset).Scan(&artists)

	// Albums.
	where, args := "", ""
	if !matchAll {
		where, args = "album LIKE ?", like
	}
	albums := h.queryAlbums(userID, where, args, "album_artist, album", albumCount, albumOffset)

	// Songs.
	var songs []models.MediaFile
	sq := h.db.Where("user_id = ?", userID)
	if !matchAll {
		sq = sq.Where("title LIKE ? OR album LIKE ? OR album_artist LIKE ? OR artists LIKE ?",
			like, like, like, like)
	}
	sq.Order("album_artist, album, disc_no, track_no").
		Limit(songCount).Offset(songOffset).Find(&songs)

	return artists, albums, songs
}
