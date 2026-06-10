package api

import (
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/fsops"
	"github.com/ultimoistante/timbre/internal/models"
)

// resolve validates a client-supplied path against the current user's root.
func (s *Server) resolve(c echo.Context, relPath string) (string, error) {
	u := auth.CurrentUser(c)
	return s.store.Resolve(u.ID, relPath)
}

func (s *Server) handleFSList(c echo.Context) error {
	abs, err := s.resolve(c, c.QueryParam("path"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	entries, err := fsops.List(abs)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "not a directory")
	}
	return c.JSON(http.StatusOK, map[string]any{
		"path":    cleanRel(c.QueryParam("path")),
		"entries": entries,
	})
}

type pathBody struct {
	Path string `json:"path"`
}

func (s *Server) handleFSMkdir(c echo.Context) error {
	var b pathBody
	if err := c.Bind(&b); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	abs, err := s.resolve(c, b.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := fsops.Mkdir(abs); err != nil {
		return err
	}
	return c.NoContent(http.StatusCreated)
}

type renameBody struct {
	Path    string `json:"path"`
	NewName string `json:"newName"`
}

func (s *Server) handleFSRename(c echo.Context) error {
	var b renameBody
	if err := c.Bind(&b); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	// newName must be a single path component (no traversal via rename).
	if b.NewName == "" || strings.ContainsAny(b.NewName, "/\\") || b.NewName == "." || b.NewName == ".." {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid new name")
	}

	srcRel := cleanRel(b.Path)
	src, err := s.resolve(c, srcRel)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	dstRel := path.Join(path.Dir(srcRel), b.NewName)
	dst, err := s.resolve(c, dstRel)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := fsops.Move(src, dst); err != nil {
		return err
	}
	u := auth.CurrentUser(c)
	s.updateMediaRelPaths(u.ID, srcRel, dstRel)
	return c.NoContent(http.StatusNoContent)
}

type srcDstBody struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

func (s *Server) handleFSMove(c echo.Context) error {
	var b srcDstBody
	if err := c.Bind(&b); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	srcRel, dstRel := cleanRel(b.Src), cleanRel(b.Dst)
	src, err := s.resolve(c, srcRel)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	dst, err := s.resolve(c, dstRel)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := fsops.Move(src, dst); err != nil {
		return err
	}
	u := auth.CurrentUser(c)
	s.updateMediaRelPaths(u.ID, srcRel, dstRel)
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleFSCopy(c echo.Context) error {
	src, dst, err := s.resolveSrcDst(c)
	if err != nil {
		return err
	}
	if err := fsops.Copy(src, dst); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) resolveSrcDst(c echo.Context) (string, string, error) {
	var b srcDstBody
	if err := c.Bind(&b); err != nil {
		return "", "", echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	src, err := s.resolve(c, b.Src)
	if err != nil {
		return "", "", echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	dst, err := s.resolve(c, b.Dst)
	if err != nil {
		return "", "", echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return src, dst, nil
}

func (s *Server) handleFSDelete(c echo.Context) error {
	var b pathBody
	if err := c.Bind(&b); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	// Refuse to delete the user root itself.
	if cleanRel(b.Path) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot delete root")
	}
	abs, err := s.resolve(c, b.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := fsops.Delete(abs); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// cleanRel normalises a client path to a root-relative slash path with no
// leading slash, for display and for deriving sibling paths.
func cleanRel(p string) string {
	clean := path.Clean("/" + strings.ReplaceAll(p, "\\", "/"))
	return strings.TrimPrefix(clean, "/")
}

// updateMediaRelPaths rewrites rel_path in media_files after a filesystem
// move or rename. Works for both single files (exact match) and directories
// (prefix match). Compatible with SQLite and PostgreSQL.
func (s *Server) updateMediaRelPaths(userID uint, srcRel, dstRel string) {
	s.db.Model(&models.MediaFile{}).
		Where("user_id = ? AND (rel_path = ? OR rel_path LIKE ?)",
			userID, srcRel, srcRel+"/%").
		Update("rel_path", gorm.Expr("? || SUBSTR(rel_path, ?)", dstRel, len(srcRel)+1))
}
