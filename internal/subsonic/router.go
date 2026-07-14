package subsonic

import "github.com/labstack/echo/v4"

// Register wires every /rest endpoint onto the group. Each method is exposed
// both bare and with the legacy ".view" suffix, on GET and POST (clients vary).
func Register(g *echo.Group, h *Handlers) {
	reg := func(name string, fn echo.HandlerFunc) {
		for _, path := range []string{"/" + name, "/" + name + ".view"} {
			g.GET(path, fn)
			g.POST(path, fn)
		}
	}

	// System.
	reg("ping", h.ping)
	reg("getLicense", h.getLicense)
	reg("getOpenSubsonicExtensions", h.getOpenSubsonicExtensions)
	reg("getUser", h.getUser)
	reg("getScanStatus", h.getScanStatus)
	reg("startScan", h.getScanStatus)

	// Browsing.
	reg("getMusicFolders", h.getMusicFolders)
	reg("getArtists", h.getArtists)
	reg("getIndexes", h.getIndexes)
	reg("getArtist", h.getArtist)
	reg("getAlbum", h.getAlbum)
	reg("getSong", h.getSong)
	reg("getGenres", h.getGenres)
	reg("getMusicDirectory", h.getMusicDirectory)

	// Lists.
	reg("getAlbumList", h.getAlbumList)
	reg("getAlbumList2", h.getAlbumList2)
	reg("getRandomSongs", h.getRandomSongs)
	reg("getNowPlaying", h.getNowPlaying)

	// Search.
	reg("search2", h.search2)
	reg("search3", h.search3)

	// Streaming.
	reg("stream", h.stream)
	reg("download", h.download)
	reg("getCoverArt", h.getCoverArt)

	// Playlists.
	reg("getPlaylists", h.getPlaylists)
	reg("getPlaylist", h.getPlaylist)
	reg("createPlaylist", h.createPlaylist)
	reg("updatePlaylist", h.updatePlaylist)
	reg("deletePlaylist", h.deletePlaylist)

	// Annotation.
	reg("scrobble", h.scrobble)
	reg("star", h.star)
	reg("unstar", h.unstar)
	reg("setRating", h.setRating)
	reg("getStarred", h.getStarred)
	reg("getStarred2", h.getStarred2)

	// Internet radio.
	reg("getInternetRadioStations", h.getInternetRadioStations)
	reg("createInternetRadioStation", h.createInternetRadioStation)
	reg("updateInternetRadioStation", h.updateInternetRadioStation)
	reg("deleteInternetRadioStation", h.deleteInternetRadioStation)
}
