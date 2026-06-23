package subsonic

import (
	"sort"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/models"
)

const ignoredArticles = "The El La Los Las Le Les"

// getMusicFolders returns the single synthetic library folder.
func (h *Handlers) getMusicFolders(c echo.Context) error {
	return Write(c, func(r *Response) {
		r.MusicFolders = &MusicFolders{
			MusicFolder: []MusicFolder{{ID: 1, Name: "Music"}},
		}
	})
}

// getArtists lists all artists, bucketed alphabetically (ID3 browsing).
func (h *Handlers) getArtists(c echo.Context) error {
	u := CurrentUser(c)
	idx := bucketArtists(h.queryArtists(u.ID))
	return Write(c, func(r *Response) {
		r.Artists = &ArtistsID3{IgnoredArticles: ignoredArticles, Index: idx}
	})
}

// getIndexes is the legacy equivalent of getArtists.
func (h *Handlers) getIndexes(c echo.Context) error {
	u := CurrentUser(c)
	idx := bucketArtists(h.queryArtists(u.ID))
	return Write(c, func(r *Response) {
		r.Indexes = &Indexes{IgnoredArticles: ignoredArticles, Index: idx}
	})
}

// getArtist returns one artist with its albums.
func (h *Handlers) getArtist(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "id"))
	if err != nil || p.Kind != KindArtist {
		return WriteError(c, ErrNotFound, "Artist not found.")
	}

	albums := h.queryAlbums(u.ID, "artist_hash = ?", p.Hash, "album", 1000, 0)
	if len(albums) == 0 {
		return WriteError(c, ErrNotFound, "Artist not found.")
	}

	out := make([]AlbumID3, len(albums))
	for i, a := range albums {
		out[i] = albumID3FromAgg(a)
	}
	return Write(c, func(r *Response) {
		r.Artist = &ArtistWithAlbums{
			ID:         ArtistID(p.Hash),
			Name:       albums[0].AlbumArtist,
			AlbumCount: len(albums),
			Album:      out,
		}
	})
}

// getAlbum returns one album with its songs.
func (h *Handlers) getAlbum(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "id"))
	if err != nil || p.Kind != KindAlbum {
		return WriteError(c, ErrNotFound, "Album not found.")
	}

	a, ok := h.albumByHash(u.ID, p.Hash)
	if !ok {
		return WriteError(c, ErrNotFound, "Album not found.")
	}
	tracks := h.albumTracks(u.ID, p.Hash)
	songs := make([]Child, len(tracks))
	for i, t := range tracks {
		songs[i] = songFromMediaFile(t)
	}
	return Write(c, func(r *Response) {
		r.Album = &AlbumWithSongs{AlbumID3: albumID3FromAgg(a), Song: songs}
	})
}

// getSong returns a single song.
func (h *Handlers) getSong(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "id"))
	if err != nil || p.Kind != KindTrack {
		return WriteError(c, ErrNotFound, "Song not found.")
	}
	mf, ok := h.trackByID(u.ID, p.UintID)
	if !ok {
		return WriteError(c, ErrNotFound, "Song not found.")
	}
	song := songFromMediaFile(mf)
	return Write(c, func(r *Response) {
		r.Song = &song
	})
}

// getGenres lists genres with song counts.
func (h *Handlers) getGenres(c echo.Context) error {
	u := CurrentUser(c)

	var rows []struct {
		Genres string
		Cnt    int
	}
	h.db.Model(&models.MediaFile{}).
		Select("genres, COUNT(*) as cnt").
		Where("user_id = ? AND genres <> ''", u.ID).
		Group("genres").
		Scan(&rows)

	counts := map[string]int{}
	for _, row := range rows {
		for _, g := range splitGenres(row.Genres) {
			counts[g] += row.Cnt
		}
	}
	names := make([]string, 0, len(counts))
	for g := range counts {
		names = append(names, g)
	}
	sort.Strings(names)

	genres := make([]Genre, len(names))
	for i, g := range names {
		genres[i] = Genre{SongCount: counts[g], Value: g}
	}
	return Write(c, func(r *Response) {
		r.Genres = &Genres{Genre: genres}
	})
}

// getMusicDirectory provides legacy folder navigation: an artist id lists its
// albums (as folders), an album id lists its songs.
func (h *Handlers) getMusicDirectory(c echo.Context) error {
	u := CurrentUser(c)
	id := param(c, "id")
	p, err := ParseID(id)
	if err != nil {
		return WriteError(c, ErrNotFound, "Directory not found.")
	}

	switch p.Kind {
	case KindArtist:
		albums := h.queryAlbums(u.ID, "artist_hash = ?", p.Hash, "album", 1000, 0)
		if len(albums) == 0 {
			return WriteError(c, ErrNotFound, "Directory not found.")
		}
		children := make([]Child, len(albums))
		for i, a := range albums {
			children[i] = dirChildFromAlbum(a)
		}
		return Write(c, func(r *Response) {
			r.Directory = &Directory{ID: id, Name: albums[0].AlbumArtist, Child: children}
		})
	case KindAlbum:
		a, ok := h.albumByHash(u.ID, p.Hash)
		if !ok {
			return WriteError(c, ErrNotFound, "Directory not found.")
		}
		tracks := h.albumTracks(u.ID, p.Hash)
		children := make([]Child, len(tracks))
		for i, t := range tracks {
			children[i] = songFromMediaFile(t)
		}
		return Write(c, func(r *Response) {
			r.Directory = &Directory{ID: id, Name: a.Album, Child: children}
		})
	default:
		return WriteError(c, ErrNotFound, "Directory not found.")
	}
}
