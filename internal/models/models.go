// Package models defines the GORM data model. Every row that belongs to a user
// carries a UserID so all queries can be scoped per user — there is no global
// shared library.
package models

import "time"

// Role enumerates user privilege levels.
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// User is an account on the server. The admin (first user, created during
// onboarding) can manage other users; each user owns an isolated media root.
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         Role      `gorm:"not null;default:user" json:"role"`
	QuotaBytes   int64     `gorm:"not null;default:0" json:"quotaBytes"` // 0 = unlimited
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// IsAdmin reports whether the user has administrative privileges.
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// MediaFile is one indexed audio track belonging to a user. It is the single
// source of truth for the library: albums and artists are derived from these
// rows via aggregate queries, scoped per user. RelPath is relative to the
// user's media root (the path understood by storage.Resolve).
type MediaFile struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"not null;uniqueIndex:idx_user_relpath;index" json:"userId"`
	RelPath string `gorm:"not null;uniqueIndex:idx_user_relpath" json:"relPath"`

	Title       string `json:"title"`
	Album       string `json:"album"`
	AlbumArtist string `json:"albumArtist"`
	Artists     string `json:"artists"` // separator-joined display string
	Genres      string `json:"genres"`
	TrackNo     int    `json:"trackNo"`
	DiscNo      int    `json:"discNo"`
	Year        int    `json:"year"`

	Duration   float64 `json:"duration"` // seconds
	Bitrate    int     `json:"bitrate"`  // bits/sec
	SampleRate int     `json:"sampleRate"`
	Container  string  `json:"container"` // file extension without dot
	SizeBytes  int64   `json:"sizeBytes"`
	ModTime    int64   `json:"modTime"` // unix seconds, for incremental scan

	TrackHash  string `gorm:"index" json:"trackHash"`
	AlbumHash  string `gorm:"index" json:"albumHash"`
	ArtistHash string `gorm:"index" json:"artistHash"`

	PlayCount  int   `json:"playCount"`
	LastPlayed int64 `json:"lastPlayed"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// Playlist is a user-curated list of tracks.
type Playlist struct {
	ID          uint            `gorm:"primaryKey" json:"id"`
	UserID      uint            `gorm:"not null;index" json:"-"`
	Name        string          `gorm:"not null" json:"name"`
	Description string          `json:"description"`
	Pinned      bool            `gorm:"default:false" json:"pinned"`
	Tracks      []PlaylistTrack `gorm:"foreignKey:PlaylistID;constraint:OnDelete:CASCADE" json:"-"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// PlaylistTrack is one entry in a playlist, ordered by Position.
type PlaylistTrack struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	PlaylistID uint `gorm:"not null;index:idx_pl_pos,priority:1" json:"playlistId"`
	TrackID    uint `gorm:"not null" json:"trackId"`
	Position   int  `gorm:"not null;default:0;index:idx_pl_pos,priority:2" json:"position"`
}

// RadioStation is a user-saved web radio (a continuous live HTTP audio stream
// identified by URL). Unlike MediaFile it has no duration, seek or queue
// semantics; it is played by proxying the upstream URL through the server.
type RadioStation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"-"`
	Name      string    `gorm:"not null" json:"name"`
	URL       string    `gorm:"not null" json:"url"` // upstream stream URL
	Genre     string    `json:"genre"`
	Homepage  string    `json:"homepage"`
	Favicon   string    `json:"favicon"` // optional logo URL
	Pinned    bool      `gorm:"default:false" json:"pinned"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AllModels returns every model for AutoMigrate.
func AllModels() []any {
	return []any{
		&User{},
		&MediaFile{},
		&Playlist{},
		&PlaylistTrack{},
		&RadioStation{},
	}
}
