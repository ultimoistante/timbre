package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// "all:" includes dirs/files whose names begin with _ or . (e.g. SvelteKit's _app/).
//
//go:embed all:frontend
var frontendFS embed.FS

// serveFrontend serves the SvelteKit build embedded in the binary.
// Real asset files are served directly; unknown paths fall back to
// index.html for SPA client-side routing.
func (s *Server) serveFrontend() {
	sub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		panic("frontend embed missing: run 'make build' first")
	}
	fileServer := http.FileServer(http.FS(sub))

	s.e.GET("/*", func(c echo.Context) error {
		// Strip leading slash: embed.FS paths never start with /.
		fsPath := strings.TrimPrefix(c.Request().URL.Path, "/")
		if fsPath == "" {
			fsPath = "index.html"
		}
		if _, err := sub.Open(fsPath); err != nil {
			// Not a real asset → SPA route, serve index.html.
			c.Request().URL.Path = "/index.html"
		}
		fileServer.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}
