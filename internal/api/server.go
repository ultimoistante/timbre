// Package api wires the HTTP server: Echo instance, middleware and route
// groups. Handlers are split across files by resource.
package api

import (
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/config"
	"github.com/ultimoistante/timbre/internal/events"
	"github.com/ultimoistante/timbre/internal/scanner"
	"github.com/ultimoistante/timbre/internal/storage"
)

// Server holds shared dependencies for all handlers.
type Server struct {
	cfg     *config.Config
	db      *gorm.DB
	jwt     *auth.Manager
	store    *storage.Store
	scanner  *scanner.Scanner
	hub      *events.Hub
	scanning sync.Map // userID -> bool, guards concurrent scans
	e        *echo.Echo
}

// New constructs the Server and registers all routes.
func New(cfg *config.Config, gdb *gorm.DB) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}))
	e.Use(middleware.Gzip())

	store := storage.New(cfg.UsersDir())
	s := &Server{
		cfg:     cfg,
		db:      gdb,
		jwt:     auth.NewManager(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL),
		store:   store,
		scanner: scanner.New(gdb, store),
		hub:     events.New(),
		e:       e,
	}

	s.registerRoutes()
	s.serveFrontend()
	return s
}

// Handler exposes the underlying Echo instance (useful for httptest).
func (s *Server) Handler() *echo.Echo { return s.e }

// Start runs the HTTP server (blocking).
func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

func (s *Server) registerRoutes() {
	api := s.e.Group("/api")

	// Public endpoints (no auth).
	api.GET("/onboarding", s.handleOnboardingStatus)
	api.POST("/onboarding", s.handleOnboarding)
	api.POST("/auth/login", s.handleLogin)
	api.POST("/auth/refresh", s.handleRefresh)
	api.GET("/healthz", func(c echo.Context) error { return c.String(200, "ok") })

	// Authenticated endpoints.
	authed := api.Group("", auth.RequireAuth(s.jwt, s.db))
	authed.GET("/me", s.handleMe)
	authed.POST("/auth/logout", s.handleLogout)

	// Filesystem CRUD (scoped to the authenticated user's media root).
	fs := authed.Group("/fs")
	fs.GET("/list", s.handleFSList)
	fs.POST("/mkdir", s.handleFSMkdir)
	fs.POST("/rename", s.handleFSRename)
	fs.POST("/move", s.handleFSMove)
	fs.POST("/copy", s.handleFSCopy)
	fs.POST("/delete", s.handleFSDelete)

	// Upload / download.
	authed.POST("/upload", s.handleUpload)
	authed.GET("/download", s.handleDownload)

	// Library scan + realtime events.
	authed.POST("/scan", s.handleScan)
	authed.GET("/events", s.handleSSE)

	// Library views (derived from MediaFile, scoped per user).
	authed.GET("/tracks", s.handleTracks)
	authed.PATCH("/tracks/:id", s.handleUpdateTrack)
	authed.GET("/albums", s.handleAlbums)
	authed.GET("/albums/:hash", s.handleAlbumTracks)
	authed.PATCH("/albums/:hash", s.handleUpdateAlbum)
	authed.GET("/albums/:hash/art", s.handleAlbumArt)
	authed.GET("/albums/:hash/art/search", s.handleSearchAlbumArt)
	authed.PUT("/albums/:hash/art", s.handleSetAlbumArt)
	authed.GET("/artists", s.handleArtists)
	authed.GET("/search", s.handleSearch)
	authed.GET("/recently-added", s.handleRecentlyAdded)

	// Playlists.
	authed.GET("/playlists", s.handleListPlaylists)
	authed.POST("/playlists", s.handleCreatePlaylist)
	authed.GET("/playlists/:id", s.handleGetPlaylist)
	authed.PUT("/playlists/:id", s.handleUpdatePlaylist)
	authed.DELETE("/playlists/:id", s.handleDeletePlaylist)
	authed.POST("/playlists/:id/tracks", s.handleAddPlaylistTracks)
	authed.DELETE("/playlists/:id/tracks/:ptId", s.handleRemovePlaylistTrack)

	// Audio streaming.
	authed.GET("/stream/:id", s.handleStream)

	// Admin endpoints.
	admin := authed.Group("/admin", auth.RequireAdmin)
	s.registerAdminRoutes(admin)
}
