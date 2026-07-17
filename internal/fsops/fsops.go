// Package fsops implements filesystem CRUD operations on already-resolved
// absolute paths. It performs no path validation itself — callers must pass
// paths obtained from storage.Resolve, which is the security boundary.
package fsops

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Entry describes a single directory listing item.
type Entry struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"isDir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// List returns the contents of a directory, directories first then files, each
// group sorted by name.
func List(absDir string) ([]Entry, error) {
	dirents, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(dirents))
	for _, d := range dirents {
		info, err := d.Info()
		if err != nil {
			continue
		}
		entries = append(entries, Entry{
			Name:    d.Name(),
			IsDir:   d.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

// Stats holds recursive totals for a directory tree.
type Stats struct {
	Files   int   `json:"files"`
	Folders int   `json:"folders"`
	Bytes   int64 `json:"bytes"`
}

// Walk computes recursive file/folder counts and total size rooted at absDir.
// absDir itself is not counted as a folder.
func Walk(absDir string) (Stats, error) {
	var st Stats
	err := filepath.Walk(absDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if p == absDir {
			return nil
		}
		if info.IsDir() {
			st.Folders++
		} else {
			st.Files++
			st.Bytes += info.Size()
		}
		return nil
	})
	return st, err
}

// Mkdir creates a directory and any missing parents.
func Mkdir(absPath string) error {
	return os.MkdirAll(absPath, 0o755)
}

// Move renames/moves src to dst. Both must already be resolved within the same
// user root.
func Move(absSrc, absDst string) error {
	if err := os.MkdirAll(filepath.Dir(absDst), 0o755); err != nil {
		return err
	}
	return os.Rename(absSrc, absDst)
}

// Delete removes a file or directory (recursively).
func Delete(absPath string) error {
	return os.RemoveAll(absPath)
}

// Copy copies a file or directory tree from src to dst.
func Copy(absSrc, absDst string) error {
	info, err := os.Stat(absSrc)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(absSrc, absDst, info)
	}
	return copyFile(absSrc, absDst, info)
}

func copyFile(src, dst string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func copyDir(src, dst string, info os.FileInfo) error {
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	dirents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, d := range dirents {
		s := filepath.Join(src, d.Name())
		t := filepath.Join(dst, d.Name())
		di, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			if err := copyDir(s, t, di); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, t, di); err != nil {
				return err
			}
		}
	}
	return nil
}
