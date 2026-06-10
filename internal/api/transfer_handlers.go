package api

import (
	"archive/zip"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
)

// handleUpload accepts one or more files via multipart form (field "file") and
// stores them under the destination directory given by the "path" query param.
func (s *Server) handleUpload(c echo.Context) error {
	destRel := c.QueryParam("path")

	// Validate destination directory.
	destDir, err := s.resolve(c, destRel)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	form, err := c.MultipartForm()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid multipart form")
	}
	files := form.File["file"]
	if len(files) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no files")
	}

	saved := make([]string, 0, len(files))
	for _, fh := range files {
		// Strip any directory components from the client filename, then
		// re-resolve to keep the file confined to the user root.
		name := filepath.Base(fh.Filename)
		abs, err := s.resolve(c, path.Join(cleanRel(destRel), name))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if err := saveUploaded(fh, abs); err != nil {
			return err
		}
		saved = append(saved, name)
	}

	u := auth.CurrentUser(c)
	s.startScanAsync(u.ID)
	return c.JSON(http.StatusCreated, map[string]any{"saved": saved})
}

func saveUploaded(fh *multipart.FileHeader, abs string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

// handleDownload streams a single file as an attachment, or a directory as a
// zip archive.
func (s *Server) handleDownload(c echo.Context) error {
	rel := c.QueryParam("path")
	abs, err := s.resolve(c, rel)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	info, err := os.Stat(abs)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	}

	if !info.IsDir() {
		return c.Attachment(abs, info.Name())
	}

	// Directory -> zip stream.
	name := info.Name()
	if name == "" || name == string(filepath.Separator) {
		name = "download"
	}
	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+name+`.zip"`)
	c.Response().WriteHeader(http.StatusOK)

	zw := zip.NewWriter(c.Response())
	defer zw.Close()

	return filepath.Walk(abs, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		relInZip, err := filepath.Rel(abs, p)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(relInZip))
		if err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
}
