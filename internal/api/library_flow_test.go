package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ultimoistante/timbre/internal/models"
	"github.com/ultimoistante/timbre/internal/scanner"
)

// seedTrack inserts a MediaFile for a user with computed hashes.
func seedTrack(t *testing.T, srv *Server, userID uint, rel, title, album, artist string, dur float64) {
	t.Helper()
	mf := models.MediaFile{
		UserID:      userID,
		RelPath:     rel,
		Title:       title,
		Album:       album,
		AlbumArtist: artist,
		Artists:     artist,
		Duration:    dur,
		AlbumHash:   scanner.AlbumHash(artist, album),
		ArtistHash:  scanner.ArtistHash(artist),
		TrackHash:   scanner.TrackHash(artist, album, title),
	}
	if err := srv.db.Create(&mf).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestLibraryAggregationAndScoping(t *testing.T) {
	srv := newTestServer(t)
	tok := onboardAdmin(t, srv)       // user 1
	_, bobTok := makeUser(t, srv, "bob") // user 2

	// User 1: two tracks same album, one other album.
	seedTrack(t, srv, 1, "a/1.mp3", "S1", "AlbumA", "ArtistX", 100)
	seedTrack(t, srv, 1, "a/2.mp3", "S2", "AlbumA", "ArtistX", 200)
	seedTrack(t, srv, 1, "b/1.mp3", "S3", "AlbumB", "ArtistY", 50)

	// User 2: one track that must never appear for user 1.
	seedTrack(t, srv, 2, "c/1.mp3", "BobSong", "BobAlbum", "BobArtist", 10)

	// Albums for user 1: expect 2 albums; AlbumA has 2 tracks, duration 300.
	rec := doJSON(t, srv, http.MethodGet, "/api/albums", "", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("albums: %d %s", rec.Code, rec.Body)
	}
	var albums []AlbumAgg
	json.Unmarshal(rec.Body.Bytes(), &albums)
	if len(albums) != 2 {
		t.Fatalf("expected 2 albums, got %d: %s", len(albums), rec.Body)
	}
	var albumA *AlbumAgg
	for i := range albums {
		if albums[i].Album == "AlbumA" {
			albumA = &albums[i]
		}
		if albums[i].Album == "BobAlbum" {
			t.Fatal("user 1 sees user 2's album — scoping broken")
		}
	}
	if albumA == nil || albumA.TrackCount != 2 || albumA.Duration != 300 {
		t.Fatalf("AlbumA aggregation wrong: %+v", albumA)
	}

	// Artists for user 1: expect 2.
	rec = doJSON(t, srv, http.MethodGet, "/api/artists", "", tok)
	var artists []ArtistAgg
	json.Unmarshal(rec.Body.Bytes(), &artists)
	if len(artists) != 2 {
		t.Fatalf("expected 2 artists, got %d", len(artists))
	}

	// Album tracks for AlbumA.
	rec = doJSON(t, srv, http.MethodGet, "/api/albums/"+albumA.AlbumHash, "", tok)
	var tracks []models.MediaFile
	json.Unmarshal(rec.Body.Bytes(), &tracks)
	if len(tracks) != 2 {
		t.Fatalf("expected 2 album tracks, got %d", len(tracks))
	}

	// Search scoped: user 1 cannot find Bob's song.
	rec = doJSON(t, srv, http.MethodGet, "/api/search?q=BobSong", "", tok)
	var found []models.MediaFile
	json.Unmarshal(rec.Body.Bytes(), &found)
	if len(found) != 0 {
		t.Fatalf("user 1 found user 2's track via search: %d", len(found))
	}

	// User 2 sees only their own track.
	rec = doJSON(t, srv, http.MethodGet, "/api/tracks", "", bobTok)
	var bobTracks []models.MediaFile
	json.Unmarshal(rec.Body.Bytes(), &bobTracks)
	if len(bobTracks) != 1 || bobTracks[0].Title != "BobSong" {
		t.Fatalf("user 2 tracks wrong: %d", len(bobTracks))
	}
}
