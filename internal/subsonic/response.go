// Package subsonic implements an OpenSubsonic-compatible API (the /rest/*
// endpoint family) as an adapter over timbre's native library, so third-party
// open-standard player apps (Symfonium, substreamer, Feishin, Amperfy, DSub,
// ...) can connect. It reuses the internal stream/storage logic and the
// per-user MediaFile library; it does not duplicate the native /api.
package subsonic

import (
	"encoding/json"
	"encoding/xml"
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	// APIVersion is the Subsonic protocol version advertised in every response.
	APIVersion = "1.16.1"
	// ServerType identifies this implementation (OpenSubsonic `type` field).
	ServerType = "timbre"
	xmlNS      = "http://subsonic.org/restapi"
)

// ServerVersion is the timbre version reported to clients. Overridable at wiring.
var ServerVersion = "0.1.0"

// Subsonic error codes (subset used here).
const (
	ErrGeneric     = 0
	ErrMissingPar  = 10
	ErrBadVersion  = 30
	ErrWrongAuth   = 40
	ErrNotAllowed  = 50
	ErrNotFound    = 70
)

// Response is the Subsonic envelope. Every payload field is a pointer with
// omitempty so only the relevant one serializes. JSON nests this under a
// top-level "subsonic-response" key (see jsonRoot).
type Response struct {
	XMLName       xml.Name `xml:"subsonic-response" json:"-"`
	Xmlns         string   `xml:"xmlns,attr" json:"-"`
	Status        string   `xml:"status,attr" json:"status"`
	Version       string   `xml:"version,attr" json:"version"`
	Type          string   `xml:"type,attr" json:"type"`
	ServerVersion string   `xml:"serverVersion,attr" json:"serverVersion"`
	OpenSubsonic  bool     `xml:"openSubsonic,attr" json:"openSubsonic"`

	Error *Error `xml:"error,omitempty" json:"error,omitempty"`

	License                *License          `xml:"license,omitempty" json:"license,omitempty"`
	MusicFolders           *MusicFolders     `xml:"musicFolders,omitempty" json:"musicFolders,omitempty"`
	Artists                *ArtistsID3       `xml:"artists,omitempty" json:"artists,omitempty"`
	Indexes                *Indexes          `xml:"indexes,omitempty" json:"indexes,omitempty"`
	Artist                 *ArtistWithAlbums `xml:"artist,omitempty" json:"artist,omitempty"`
	Album                  *AlbumWithSongs   `xml:"album,omitempty" json:"album,omitempty"`
	Song                   *Child            `xml:"song,omitempty" json:"song,omitempty"`
	Directory              *Directory        `xml:"directory,omitempty" json:"directory,omitempty"`
	Genres                 *Genres           `xml:"genres,omitempty" json:"genres,omitempty"`
	AlbumList              *AlbumList        `xml:"albumList,omitempty" json:"albumList,omitempty"`
	AlbumList2             *AlbumList2       `xml:"albumList2,omitempty" json:"albumList2,omitempty"`
	RandomSongs            *Songs            `xml:"randomSongs,omitempty" json:"randomSongs,omitempty"`
	NowPlaying             *NowPlaying       `xml:"nowPlaying,omitempty" json:"nowPlaying,omitempty"`
	SearchResult2          *SearchResult2    `xml:"searchResult2,omitempty" json:"searchResult2,omitempty"`
	SearchResult3          *SearchResult3    `xml:"searchResult3,omitempty" json:"searchResult3,omitempty"`
	Playlists              *Playlists        `xml:"playlists,omitempty" json:"playlists,omitempty"`
	Playlist               *PlaylistWithSongs `xml:"playlist,omitempty" json:"playlist,omitempty"`
	Starred                *Starred          `xml:"starred,omitempty" json:"starred,omitempty"`
	Starred2               *Starred2         `xml:"starred2,omitempty" json:"starred2,omitempty"`
	User                   *SubsonicUser     `xml:"user,omitempty" json:"user,omitempty"`
	ScanStatus             *ScanStatus       `xml:"scanStatus,omitempty" json:"scanStatus,omitempty"`
	OpenSubsonicExtensions []OSExtension     `xml:"openSubsonicExtension,omitempty" json:"openSubsonicExtensions,omitempty"`
	InternetRadioStations  *InternetRadioStations `xml:"internetRadioStations,omitempty" json:"internetRadioStations,omitempty"`
}

// Error is the Subsonic error element.
type Error struct {
	Code    int    `xml:"code,attr" json:"code"`
	Message string `xml:"message,attr" json:"message"`
}

// License is reported as always valid (self-hosted, no licensing).
type License struct {
	Valid bool `xml:"valid,attr" json:"valid"`
}

// MusicFolders wraps the (single, synthetic) music folder list.
type MusicFolders struct {
	MusicFolder []MusicFolder `xml:"musicFolder" json:"musicFolder"`
}

// MusicFolder is one top-level library folder.
type MusicFolder struct {
	ID   int    `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

// ArtistsID3 is the getArtists payload (ID3-style browsing).
type ArtistsID3 struct {
	IgnoredArticles string     `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Index           []IndexID3 `xml:"index" json:"index"`
}

// Indexes is the legacy getIndexes payload (same letter-bucket shape).
type Indexes struct {
	IgnoredArticles string     `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	LastModified    int64      `xml:"lastModified,attr" json:"lastModified"`
	Index           []IndexID3 `xml:"index" json:"index"`
}

// IndexID3 is one alphabetical bucket of artists.
type IndexID3 struct {
	Name   string      `xml:"name,attr" json:"name"`
	Artist []ArtistID3 `xml:"artist" json:"artist"`
}

// ArtistID3 is an artist in ID3 browsing.
type ArtistID3 struct {
	ID         string `xml:"id,attr" json:"id"`
	Name       string `xml:"name,attr" json:"name"`
	CoverArt   string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
}

// ArtistWithAlbums is the getArtist payload.
type ArtistWithAlbums struct {
	ID         string     `xml:"id,attr" json:"id"`
	Name       string     `xml:"name,attr" json:"name"`
	CoverArt   string     `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	AlbumCount int        `xml:"albumCount,attr" json:"albumCount"`
	Album      []AlbumID3 `xml:"album" json:"album"`
}

// AlbumID3 is an album in ID3 browsing / album lists.
type AlbumID3 struct {
	ID        string `xml:"id,attr" json:"id"`
	Name      string `xml:"name,attr" json:"name"`
	Artist    string `xml:"artist,attr" json:"artist"`
	ArtistID  string `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	CoverArt  string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	SongCount int    `xml:"songCount,attr" json:"songCount"`
	Duration  int    `xml:"duration,attr" json:"duration"`
	Year      int    `xml:"year,attr,omitempty" json:"year,omitempty"`
}

// AlbumWithSongs is the getAlbum payload (album metadata + its songs).
type AlbumWithSongs struct {
	AlbumID3
	Song []Child `xml:"song" json:"song"`
}

// Child is the shared song/directory entry used across most endpoints.
type Child struct {
	ID          string `xml:"id,attr" json:"id"`
	Parent      string `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	IsDir       bool   `xml:"isDir,attr" json:"isDir"`
	Title       string `xml:"title,attr" json:"title"`
	Album       string `xml:"album,attr,omitempty" json:"album,omitempty"`
	Artist      string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Track       int    `xml:"track,attr,omitempty" json:"track,omitempty"`
	Year        int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre       string `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	CoverArt    string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Size        int64  `xml:"size,attr,omitempty" json:"size,omitempty"`
	ContentType string `xml:"contentType,attr,omitempty" json:"contentType,omitempty"`
	Suffix      string `xml:"suffix,attr,omitempty" json:"suffix,omitempty"`
	Duration    int    `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	BitRate     int    `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	DiscNumber  int    `xml:"discNumber,attr,omitempty" json:"discNumber,omitempty"`
	AlbumID     string `xml:"albumId,attr,omitempty" json:"albumId,omitempty"`
	ArtistID    string `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	Type        string `xml:"type,attr,omitempty" json:"type,omitempty"`
	PlayCount   int    `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	Created     string `xml:"created,attr,omitempty" json:"created,omitempty"`
	Starred     string `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	MediaType   string `xml:"mediaType,attr,omitempty" json:"mediaType,omitempty"` // OpenSubsonic
}

// Directory is the legacy getMusicDirectory payload.
type Directory struct {
	ID    string  `xml:"id,attr" json:"id"`
	Name  string  `xml:"name,attr" json:"name"`
	Child []Child `xml:"child" json:"child"`
}

// Genres is the getGenres payload.
type Genres struct {
	Genre []Genre `xml:"genre" json:"genre"`
}

// Genre is one genre with counts.
type Genre struct {
	SongCount  int    `xml:"songCount,attr" json:"songCount"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
	Value      string `xml:",chardata" json:"value"`
}

// AlbumList2 is the getAlbumList2 payload (ID3 albums).
type AlbumList2 struct {
	Album []AlbumID3 `xml:"album" json:"album"`
}

// AlbumList is the legacy getAlbumList payload (directory-style albums).
type AlbumList struct {
	Album []Child `xml:"album" json:"album"`
}

// Songs wraps a flat song list (randomSongs).
type Songs struct {
	Song []Child `xml:"song" json:"song"`
}

// NowPlaying is the getNowPlaying payload (always empty here).
type NowPlaying struct {
	Entry []Child `xml:"entry" json:"entry"`
}

// SearchResult3 is the search3 payload (ID3).
type SearchResult3 struct {
	Artist []ArtistID3 `xml:"artist" json:"artist,omitempty"`
	Album  []AlbumID3  `xml:"album" json:"album,omitempty"`
	Song   []Child     `xml:"song" json:"song,omitempty"`
}

// SearchResult2 is the legacy search2 payload.
type SearchResult2 struct {
	Artist []ArtistID3 `xml:"artist" json:"artist,omitempty"`
	Album  []Child     `xml:"album" json:"album,omitempty"`
	Song   []Child     `xml:"song" json:"song,omitempty"`
}

// Playlists wraps the playlist list.
type Playlists struct {
	Playlist []PlaylistDTO `xml:"playlist" json:"playlist"`
}

// PlaylistDTO is a playlist summary.
type PlaylistDTO struct {
	ID        string `xml:"id,attr" json:"id"`
	Name      string `xml:"name,attr" json:"name"`
	Comment   string `xml:"comment,attr,omitempty" json:"comment,omitempty"`
	Owner     string `xml:"owner,attr,omitempty" json:"owner,omitempty"`
	SongCount int    `xml:"songCount,attr" json:"songCount"`
	Duration  int    `xml:"duration,attr" json:"duration"`
	Public    bool   `xml:"public,attr" json:"public"`
	Created   string `xml:"created,attr,omitempty" json:"created,omitempty"`
	Changed   string `xml:"changed,attr,omitempty" json:"changed,omitempty"`
}

// PlaylistWithSongs is the getPlaylist payload.
type PlaylistWithSongs struct {
	PlaylistDTO
	Entry []Child `xml:"entry" json:"entry"`
}

// Starred2 is the getStarred2 payload (ID3).
type Starred2 struct {
	Artist []ArtistID3 `xml:"artist" json:"artist,omitempty"`
	Album  []AlbumID3  `xml:"album" json:"album,omitempty"`
	Song   []Child     `xml:"song" json:"song,omitempty"`
}

// Starred is the legacy getStarred payload.
type Starred struct {
	Artist []ArtistID3 `xml:"artist" json:"artist,omitempty"`
	Album  []Child     `xml:"album" json:"album,omitempty"`
	Song   []Child     `xml:"song" json:"song,omitempty"`
}

// SubsonicUser is the getUser payload reporting capability roles.
type SubsonicUser struct {
	Username     string `xml:"username,attr" json:"username"`
	AdminRole    bool   `xml:"adminRole,attr" json:"adminRole"`
	SettingsRole bool   `xml:"settingsRole,attr" json:"settingsRole"`
	DownloadRole bool   `xml:"downloadRole,attr" json:"downloadRole"`
	UploadRole   bool   `xml:"uploadRole,attr" json:"uploadRole"`
	PlaylistRole bool   `xml:"playlistRole,attr" json:"playlistRole"`
	CoverArtRole bool   `xml:"coverArtRole,attr" json:"coverArtRole"`
	StreamRole   bool   `xml:"streamRole,attr" json:"streamRole"`
	ScrobblingEnabled bool `xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	Folder       []int  `xml:"folder" json:"folder"`
}

// ScanStatus is the get/startScan payload.
type ScanStatus struct {
	Scanning bool `xml:"scanning,attr" json:"scanning"`
	Count    int  `xml:"count,attr" json:"count"`
}

// InternetRadioStations wraps the getInternetRadioStations payload.
type InternetRadioStations struct {
	InternetRadioStation []InternetRadioStation `xml:"internetRadioStation" json:"internetRadioStation"`
}

// InternetRadioStation is one saved web radio (Timbre's RadioStation, adapted).
type InternetRadioStation struct {
	ID          string `xml:"id,attr" json:"id"`
	Name        string `xml:"name,attr" json:"name"`
	StreamURL   string `xml:"streamUrl,attr" json:"streamUrl"`
	HomePageURL string `xml:"homePageUrl,attr,omitempty" json:"homePageUrl,omitempty"`
}

// OSExtension advertises one OpenSubsonic extension.
type OSExtension struct {
	Name     string `xml:"name,attr" json:"name"`
	Versions []int  `xml:"versions" json:"versions"`
}

// jsonRoot nests the envelope under "subsonic-response" for JSON/JSONP.
type jsonRoot struct {
	SR Response `json:"subsonic-response"`
}

// newResponse builds the common envelope base.
func newResponse() Response {
	return Response{
		Xmlns:         xmlNS,
		Status:        "ok",
		Version:       APIVersion,
		Type:          ServerType,
		ServerVersion: ServerVersion,
		OpenSubsonic:  true,
	}
}

// Write emits a success response, applying mutate to fill the payload.
func Write(c echo.Context, mutate func(*Response)) error {
	r := newResponse()
	if mutate != nil {
		mutate(&r)
	}
	return encode(c, &r)
}

// WriteError emits a Subsonic error (HTTP 200, status="failed").
func WriteError(c echo.Context, code int, msg string) error {
	r := newResponse()
	r.Status = "failed"
	r.Error = &Error{Code: code, Message: msg}
	return encode(c, &r)
}

// encode serializes the envelope as XML (default), JSON or JSONP per the f param.
func encode(c echo.Context, r *Response) error {
	switch param(c, "f") {
	case "json":
		return c.JSON(http.StatusOK, jsonRoot{SR: *r})
	case "jsonp":
		cb := param(c, "callback")
		if cb == "" {
			cb = "callback"
		}
		b, err := json.Marshal(jsonRoot{SR: *r})
		if err != nil {
			return err
		}
		return c.Blob(http.StatusOK, "application/javascript; charset=utf-8",
			append(append([]byte(cb+"("), b...), ");"...))
	default:
		b, err := xml.Marshal(r)
		if err != nil {
			return err
		}
		return c.Blob(http.StatusOK, "application/xml; charset=utf-8",
			append([]byte(xml.Header), b...))
	}
}
