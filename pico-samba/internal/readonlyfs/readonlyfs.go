// Package readonlyfs implements read-only VFS for go-smb2 with symlink rejection.
package readonlyfs

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"github.com/macos-fuse-t/go-smb2/vfs"
	"golang.org/x/sys/unix"
)

var errReadOnly = errors.New("read-only filesystem")
var errSymlinkForbidden = errors.New("symlinks are forbidden")
var errAccessDenied = errors.New("access denied")

type openFile struct {
	path   string
	file   *os.File
	handle vfs.VfsHandle
	isDir  bool
	dirPos int
}

type ReadOnlyFS struct {
	rootPath  string
	openFiles sync.Map
}

func New(rootPath string) *ReadOnlyFS {
	return &ReadOnlyFS{rootPath: rootPath}
}

func (fs *ReadOnlyFS) absPath(p string) string {
	return path.Join(fs.rootPath, p)
}

func (fs *ReadOnlyFS) getPath(handle vfs.VfsHandle) (string, error) {
	v, ok := fs.openFiles.Load(handle)
	if !ok {
		return "", fmt.Errorf("bad handle")
	}
	return v.(*openFile).path, nil
}

func (fs *ReadOnlyFS) getOpen(handle vfs.VfsHandle) (*openFile, error) {
	v, ok := fs.openFiles.Load(handle)
	if !ok {
		return nil, fmt.Errorf("bad handle")
	}
	return v.(*openFile), nil
}

func randHandle() vfs.VfsHandle {
	var b [8]byte
	rand.Read(b[:])
	return vfs.VfsHandle(binary.LittleEndian.Uint64(b[:]))
}

func fileInfoToAttr(info os.FileInfo) (*vfs.Attributes, error) {
	sysStat, ok := vfs.CompatStat(info)
	if !ok {
		return nil, fmt.Errorf("failed to convert to syscall.Stat_t")
	}
	a := vfs.Attributes{}
	a.SetInodeNumber(sysStat.Ino)
	a.SetSizeBytes(uint64(info.Size()))
	a.SetDiskSizeBytes(uint64(sysStat.Blocks * 512))
	a.SetUnixMode(uint32(info.Mode()))
	a.SetPermissions(vfs.NewPermissionsFromMode(uint32(info.Mode().Perm())))
	a.SetAccessTime(sysStat.Atime)
	a.SetLastDataModificationTime(info.ModTime())
	a.SetBirthTime(sysStat.Btime)
	a.SetLastStatusChangeTime(sysStat.Ctime)
	if info.IsDir() {
		a.SetFileType(vfs.FileTypeDirectory)
	} else if info.Mode()&os.ModeSymlink != 0 {
		a.SetFileType(vfs.FileTypeSymlink)
	} else {
		a.SetFileType(vfs.FileTypeRegularFile)
	}
	return &a, nil
}

func (fs *ReadOnlyFS) GetAttr(handle vfs.VfsHandle) (*vfs.Attributes, error) {
	p := fs.rootPath
	if handle != 0 {
		var err error
		if p, err = fs.getPath(handle); err != nil {
			return nil, err
		}
	}
	info, err := os.Lstat(p)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errSymlinkForbidden
	}
	return fileInfoToAttr(info)
}

func (fs *ReadOnlyFS) SetAttr(handle vfs.VfsHandle, _ *vfs.Attributes) (*vfs.Attributes, error) {
	return nil, errReadOnly
}

func (fs *ReadOnlyFS) StatFS(handle vfs.VfsHandle) (*vfs.FSAttributes, error) {
	var statfs unix.Statfs_t
	if err := unix.Statfs(fs.rootPath, &statfs); err != nil {
		return nil, err
	}
	a := vfs.FSAttributes{}
	a.SetAvailableBlocks(uint64(statfs.Bavail))
	a.SetBlockSize(uint64(statfs.Bsize))
	a.SetBlocks(uint64(statfs.Blocks))
	a.SetFiles(uint64(statfs.Files))
	a.SetFreeBlocks(uint64(statfs.Bfree))
	a.SetFreeFiles(uint64(statfs.Ffree))
	a.SetIOSize(uint64(statfs.Bsize))
	return &a, nil
}

func (fs *ReadOnlyFS) FSync(handle vfs.VfsHandle) error {
	open, err := fs.getOpen(handle)
	if err != nil {
		return err
	}
	return open.file.Sync()
}

func (fs *ReadOnlyFS) Flush(handle vfs.VfsHandle) error {
	return nil
}

func (fs *ReadOnlyFS) Open(p string, flags int, mode int) (vfs.VfsHandle, error) {
	fullPath := fs.absPath(p)
	info, err := os.Lstat(fullPath)
	if err != nil {
		return 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 0, errSymlinkForbidden
	}
	if info.IsDir() {
		f, err := os.Open(fullPath)
		if err != nil {
			return 0, err
		}
		h := randHandle()
		fs.openFiles.Store(h, &openFile{path: fullPath, file: f, handle: h, isDir: true})
		return h, nil
	}
	if flags&(os.O_WRONLY|os.O_RDWR) != 0 {
		return 0, errReadOnly
	}
	f, err := os.Open(fullPath)
	if err != nil {
		return 0, err
	}
	h := randHandle()
	fs.openFiles.Store(h, &openFile{path: fullPath, file: f, handle: h})
	return h, nil
}

func (fs *ReadOnlyFS) Close(handle vfs.VfsHandle) error {
	open, err := fs.getOpen(handle)
	if err != nil {
		return err
	}
	open.file.Close()
	fs.openFiles.Delete(handle)
	return nil
}

func (fs *ReadOnlyFS) Lookup(handle vfs.VfsHandle, name string) (*vfs.Attributes, error) {
	p := fs.rootPath
	if handle != 0 {
		var err error
		if p, err = fs.getPath(handle); err != nil {
			return nil, err
		}
	}
	info, err := os.Lstat(path.Join(p, name))
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errSymlinkForbidden
	}
	return fileInfoToAttr(info)
}

func (fs *ReadOnlyFS) Mkdir(p string, mode int) (*vfs.Attributes, error) {
	return nil, errReadOnly
}

func (fs *ReadOnlyFS) Read(handle vfs.VfsHandle, buf []byte, offset uint64, flags int) (int, error) {
	open, err := fs.getOpen(handle)
	if err != nil {
		return 0, err
	}
	return open.file.ReadAt(buf, int64(offset))
}

func (fs *ReadOnlyFS) Write(handle vfs.VfsHandle, buf []byte, offset uint64, flags int) (int, error) {
	return 0, errReadOnly
}

func (fs *ReadOnlyFS) OpenDir(p string) (vfs.VfsHandle, error) {
	fullPath := fs.absPath(p)
	info, err := os.Lstat(fullPath)
	if err != nil {
		return 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 0, errSymlinkForbidden
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("not a directory")
	}
	f, err := os.Open(fullPath)
	if err != nil {
		return 0, err
	}
	h := randHandle()
	fs.openFiles.Store(h, &openFile{path: fullPath, file: f, handle: h, isDir: true})
	return h, nil
}

func (fs *ReadOnlyFS) ReadDir(handle vfs.VfsHandle, pos int, maxEntries int) ([]vfs.DirInfo, error) {
	open, err := fs.getOpen(handle)
	if err != nil {
		return nil, err
	}
	if !open.isDir {
		return nil, fmt.Errorf("not a directory")
	}
	if pos != 0 {
		open.file.Seek(0, 0)
		open.dirPos = 0
	}
	var results []vfs.DirInfo
	if open.dirPos == 0 {
		attrs, err := fs.GetAttr(handle)
		if err != nil {
			return nil, err
		}
		results = append(results,
			vfs.DirInfo{Name: ".", Attributes: *attrs},
			vfs.DirInfo{Name: "..", Attributes: *attrs},
		)
	}
	entries, err := open.file.ReadDir(maxEntries)
	if err != nil && (err != io.EOF || open.dirPos != 0) {
		return nil, err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		attrs, err := fileInfoToAttr(info)
		if err != nil {
			continue
		}
		results = append(results, vfs.DirInfo{Name: entry.Name(), Attributes: *attrs})
	}
	open.dirPos = 1
	return results, nil
}

func (fs *ReadOnlyFS) Readlink(handle vfs.VfsHandle) (string, error) {
	return "", errSymlinkForbidden
}

func (fs *ReadOnlyFS) Unlink(handle vfs.VfsHandle) error {
	return errReadOnly
}

func (fs *ReadOnlyFS) Truncate(handle vfs.VfsHandle, length uint64) error {
	return errReadOnly
}

func (fs *ReadOnlyFS) Rename(from vfs.VfsHandle, to string, flags int) error {
	return errReadOnly
}

func (fs *ReadOnlyFS) Symlink(targetHandle vfs.VfsHandle, source string, flag int) (*vfs.Attributes, error) {
	return nil, errReadOnly
}

func (fs *ReadOnlyFS) Link(_, _ vfs.VfsNode, _ string) (*vfs.Attributes, error) {
	return nil, errReadOnly
}

func (fs *ReadOnlyFS) Listxattr(handle vfs.VfsHandle) ([]string, error) {
	return nil, nil
}

func (fs *ReadOnlyFS) Getxattr(handle vfs.VfsHandle, key string, val []byte) (int, error) {
	return 0, nil
}

func (fs *ReadOnlyFS) Setxattr(handle vfs.VfsHandle, key string, val []byte) error {
	return errReadOnly
}

func (fs *ReadOnlyFS) Removexattr(handle vfs.VfsHandle, key string) error {
	return errReadOnly
}
