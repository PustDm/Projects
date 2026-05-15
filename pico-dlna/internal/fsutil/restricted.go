package fsutil

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ErrSymlinkForbidden is returned when a symlink is encountered.
var ErrSymlinkForbidden = fs.ErrInvalid

// NoSymlinkFS implements fs.FS and forbids symlink traversal.
// It wraps a directory and ensures no symlinks are followed when
// listing directories or opening files.
type NoSymlinkFS struct {
	root string
}

// NewNoSymlinkFS creates a filesystem that rejects symlinks.
func NewNoSymlinkFS(root string) (*NoSymlinkFS, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, &fs.PathError{Op: "open", Path: root, Err: ErrSymlinkForbidden}
	}
	return &NoSymlinkFS{root: abs}, nil
}

// Open opens the named file. Symlinks are rejected.
func (f *NoSymlinkFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	fullPath, err := f.resolveAndCheck(name)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if info.IsDir() {
		file.Close()
		return &noSymlinkDir{path: fullPath, fs: f}, nil
	}
	return &noSymlinkFile{File: file, path: fullPath}, nil
}

// resolveAndCheck returns the full path and ensures no path component is a symlink.
func (f *NoSymlinkFS) resolveAndCheck(name string) (string, error) {
	clean := path.Clean(name)
	if clean == "." || clean == "" {
		return f.root, nil
	}
	parts := strings.Split(clean, "/")
	current := f.root
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", &fs.PathError{Op: "open", Path: name, Err: err}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", &fs.PathError{Op: "open", Path: name, Err: ErrSymlinkForbidden}
		}
	}
	return current, nil
}

// fullPath returns the full filesystem path for a relative name.
func (f *NoSymlinkFS) fullPath(name string) string {
	if name == "." || name == "" {
		return f.root
	}
	return filepath.Join(f.root, filepath.FromSlash(path.Clean(name)))
}

type noSymlinkFile struct {
	*os.File
	path string
}

func (f *noSymlinkFile) Stat() (fs.FileInfo, error) {
	return f.File.Stat()
}

type noSymlinkDir struct {
	path string
	fs   *NoSymlinkFS
}

func (d *noSymlinkDir) ReadDir(n int) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(d.path)
	if err != nil {
		return nil, err
	}
	var result []fs.DirEntry
	for _, e := range entries {
		fullPath := filepath.Join(d.path, e.Name())
		info, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		result = append(result, e)
	}
	if n > 0 && len(result) > n {
		result = result[:n]
	}
	return result, nil
}

func (d *noSymlinkDir) Stat() (fs.FileInfo, error) {
	return os.Stat(d.path)
}

func (d *noSymlinkDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.path, Err: fs.ErrInvalid}
}

func (d *noSymlinkDir) Close() error {
	return nil
}
