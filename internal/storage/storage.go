// Package storage is the single security boundary for user file access. Every
// path supplied by a client passes through Resolve, which confines it to the
// requesting user's isolated media root and rejects path-traversal and
// symlink-escape attempts. Handlers must never build filesystem paths by hand.
package storage

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrEscape is returned when a path would resolve outside the user's root.
var ErrEscape = errors.New("path escapes user root")

// Store resolves and validates per-user file paths.
type Store struct {
	usersDir string
}

// New creates a Store rooted at usersDir (the parent of every per-user root).
func New(usersDir string) *Store {
	return &Store{usersDir: usersDir}
}

// UserRoot returns the absolute media root for a user id.
func (s *Store) UserRoot(userID uint) string {
	return filepath.Join(s.usersDir, strconv.FormatUint(uint64(userID), 10))
}

// EnsureUserRoot creates the user's media root if it does not exist.
func (s *Store) EnsureUserRoot(userID uint) error {
	return os.MkdirAll(s.UserRoot(userID), 0o755)
}

// Resolve validates relPath against the user's root and returns the absolute
// path. relPath is always interpreted relative to the user root: a leading
// slash is anchored to the root, internal ".." that stays inside is allowed,
// and any ".." that would escape — or a symlink pointing outside — is rejected.
//
// The target need not exist (mkdir/upload create it); symlink resolution is
// applied to the longest existing ancestor.
func (s *Store) Resolve(userID uint, relPath string) (string, error) {
	root := s.UserRoot(userID)

	// Resolve symlinks on the root itself so comparisons use canonical paths.
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}

	// Normalise the requested path. Absolute paths are anchored to the user
	// root (a leading slash means "root", never the real FS root), so they can
	// never escape. Relative paths are cleaned and any ".." that would climb
	// above the root is rejected outright rather than silently clamped.
	slash := filepath.ToSlash(relPath)
	var clean string
	if strings.HasPrefix(slash, "/") {
		clean = strings.TrimPrefix(path.Clean(slash), "/")
	} else {
		clean = path.Clean(slash)
		if clean == ".." || strings.HasPrefix(clean, "../") {
			return "", ErrEscape
		}
		if clean == "." {
			clean = ""
		}
	}
	candidate := filepath.Join(realRoot, filepath.FromSlash(clean))

	// Lexical containment check (defence in depth before touching the FS).
	if !within(realRoot, candidate) {
		return "", ErrEscape
	}

	// Resolve symlinks on the existing portion and re-check containment.
	resolved, err := resolveExisting(candidate)
	if err != nil {
		return "", err
	}
	if !within(realRoot, resolved) {
		return "", ErrEscape
	}
	return resolved, nil
}

// within reports whether p is root or a descendant of root.
func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// resolveExisting evaluates symlinks on the longest existing prefix of p and
// re-appends the non-existing remainder, so paths to not-yet-created files can
// still be validated against symlink escapes in their existing ancestors.
func resolveExisting(p string) (string, error) {
	remaining := ""
	cur := p
	for {
		resolved, err := filepath.EvalSymlinks(cur)
		if err == nil {
			if remaining == "" {
				return resolved, nil
			}
			return filepath.Join(resolved, remaining), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", err
		}
		remaining = filepath.Join(filepath.Base(cur), remaining)
		cur = parent
	}
}
