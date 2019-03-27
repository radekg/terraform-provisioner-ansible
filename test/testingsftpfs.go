package test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/pkg/sftp"
)

// NewTestingSFTPFS returns a Hanlders object with the test handlers.
func NewTestingSFTPFS(t *testing.T, serverConfig *TestingSSHServerConfig, chanNotifications chan interface{}) sftp.Handlers {
	root := &root{
		t:                 t,
		serverConfig:      serverConfig,
		chanNotifications: chanNotifications,
		files:             make(map[string]*memFile),
	}
	root.memFile = newMemFile("/", true)
	return sftp.Handlers{
		FileGet:  root,
		FilePut:  root,
		FileCmd:  root,
		FileList: root,
	}
}

// Example Handlers
func (fs *root) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	fs.t.Logf("[%s][Reader]: Reading file: %v", fs.serverConfig.ServerID, r)
	if fs.mockErr != nil {
		return nil, fs.mockErr
	}
	_ = r.WithContext(r.Context()) // initialize context for deadlock testing
	fs.filesLock.Lock()
	defer fs.filesLock.Unlock()
	file, err := fs.fetch(r.Filepath)
	if err != nil {
		fs.t.Logf("[%s][Reader]: Read error %v for %v", fs.serverConfig.ServerID, err, r)
		return nil, err
	}
	if file.symlink != "" {
		file, err = fs.fetch(file.symlink)
		if err != nil {
			return nil, err
		}
	}
	return file.ReaderAt()
}

func (fs *root) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	fs.t.Logf("[%s][Writer]: Writing file: %v", fs.serverConfig.ServerID, r)
	if fs.mockErr != nil {
		return nil, fs.mockErr
	}
	_ = r.WithContext(r.Context()) // initialize context for deadlock testing
	fs.filesLock.Lock()
	defer fs.filesLock.Unlock()
	file, err := fs.fetch(r.Filepath)
	if err == os.ErrNotExist {
		dir, err := fs.fetch(filepath.Dir(r.Filepath))
		if err != nil {
			fs.t.Logf("[%s][Writer]: Write error %v for %v", fs.serverConfig.ServerID, err, r)
			return nil, err
		}
		if !dir.isdir {
			fs.t.Logf("[%s][Writer]: Write error %v for %v", fs.serverConfig.ServerID, err, r)
			return nil, os.ErrInvalid
		}
		file = newMemFile(r.Filepath, false)
		fs.files[r.Filepath] = file
	}

	go func(fsq *root, rq *sftp.Request, cfile *memFile) {
		if fsq == nil || rq == nil || cfile == nil {
			if fsq == nil {
				fs.t.Log("[---][Writer]: Filesystem is nil on finished write!")
			} else {
				fs.t.Logf("[%s][Writer]: Request or memfile is nil on finished write!", fsq.serverConfig.ServerID)
			}
			return
		}
		<-rq.Context().Done()
		go func() {
			fsq.chanNotifications <- NotificationFileWritten{
				MemFile: cfile,
			}
		}()

	}(fs, r, file)

	return file.WriterAt()
}

func (fs *root) Filecmd(r *sftp.Request) error {
	if fs.mockErr != nil {
		return fs.mockErr
	}
	_ = r.WithContext(r.Context()) // initialize context for deadlock testing
	fs.filesLock.Lock()
	defer fs.filesLock.Unlock()
	switch r.Method {
	case "Setstat":
		fs.t.Logf("[%s][Cmder]: Setstat %v", fs.serverConfig.ServerID, r)
		return nil
	case "Rename":
		fs.t.Logf("[%s][Cmder]: Rename %v", fs.serverConfig.ServerID, r)
		file, err := fs.fetch(r.Filepath)
		if err != nil {
			fs.t.Logf("[%s][Cmder]: Rename error %v for %v", fs.serverConfig.ServerID, err, r)
			return err
		}
		if _, ok := fs.files[r.Target]; ok {
			err := &os.LinkError{Op: "rename", Old: r.Filepath, New: r.Target,
				Err: fmt.Errorf("dest file exists")}
			fs.t.Logf("[%s][Cmder]: Rename error %v for %v", fs.serverConfig.ServerID, *err, r)
			return err
		}

		file.name = r.Target
		fs.files[r.Target] = file
		delete(fs.files, r.Filepath)
		go func() {
			fs.chanNotifications <- NotificationFileRenamed{
				MemFile:    file,
				SourcePath: r.Filepath,
			}
		}()

	case "Rmdir", "Remove":
		fs.t.Logf("[%s][Cmder]: Remove %v", fs.serverConfig.ServerID, r)
		file, err := fs.fetch(filepath.Dir(r.Filepath))
		if err != nil {
			fs.t.Logf("[%s][Cmder]: Remove error %v for %v", fs.serverConfig.ServerID, err, r)
			return err
		}
		delete(fs.files, r.Filepath)
		go func() {
			fs.chanNotifications <- NotificationDirectoryDeleted{
				MemFile: file,
			}
		}()
	case "Mkdir":
		fs.t.Logf("[%s][Cmder]: Mkdir %v", fs.serverConfig.ServerID, r)
		_, err := fs.fetch(filepath.Dir(r.Filepath))
		if err != nil {
			fs.t.Logf("[%s][Cmder]: Mkdir error %v for %v", fs.serverConfig.ServerID, err, r)
			return err
		}
		fs.files[r.Filepath] = newMemFile(r.Filepath, true)
		go func() {
			fs.chanNotifications <- NotificationDirectoryCreated{
				MemFile: fs.files[r.Filepath],
			}
		}()
	case "Symlink":
		fs.t.Logf("[%s][Cmder]: Symlink %v", fs.serverConfig.ServerID, r)
		_, err := fs.fetch(r.Filepath)
		if err != nil {
			fs.t.Logf("[%s][Cmder]: Symlink error %v for %v", fs.serverConfig.ServerID, err, r)
			return err
		}
		link := newMemFile(r.Target, false)
		link.symlink = r.Filepath
		fs.files[r.Target] = link
		go func() {
			fs.chanNotifications <- NotificationSymlinkCreated{
				MemFile: link,
			}
		}()
	}
	return nil
}

type listerat []os.FileInfo

// Modeled after strings.Reader's ReadAt() implementation
func (f listerat) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	var n int
	if offset >= int64(len(f)) {
		return 0, io.EOF
	}
	n = copy(ls, f[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}

func (fs *root) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	if fs.mockErr != nil {
		return nil, fs.mockErr
	}
	_ = r.WithContext(r.Context()) // initialize context for deadlock testing
	fs.filesLock.Lock()
	defer fs.filesLock.Unlock()

	file, err := fs.fetch(r.Filepath)
	if err != nil {
		return nil, err
	}

	switch r.Method {
	case "List":
		fs.t.Logf("[%s][Lister]: List %v", fs.serverConfig.ServerID, r)
		if !file.IsDir() {
			fs.t.Logf("[%s][Lister]: Not directory %v", fs.serverConfig.ServerID, file)
			return nil, syscall.ENOTDIR
		}
		orderedNames := []string{}
		for fn := range fs.files {
			if filepath.Dir(fn) == r.Filepath {
				orderedNames = append(orderedNames, fn)
			}
		}
		sort.Strings(orderedNames)
		list := make([]os.FileInfo, len(orderedNames))
		for i, fn := range orderedNames {
			list[i] = fs.files[fn]
		}
		fs.t.Logf("[%s][Lister]: %s contains: %v", fs.serverConfig.ServerID, r.Filepath, orderedNames)
		return listerat(list), nil
	case "Stat":
		fs.t.Logf("[%s][Lister]: Stat %v", fs.serverConfig.ServerID, r)
		return listerat([]os.FileInfo{file}), nil
	case "Readlink":
		fs.t.Logf("[%s][Lister]: Readlink %v", fs.serverConfig.ServerID, r)
		if file.symlink != "" {
			file, err = fs.fetch(file.symlink)
			if err != nil {
				return nil, err
			}
		}
		return listerat([]os.FileInfo{file}), nil
	}
	return nil, nil
}

// In memory file-system-y thing that the Hanlders live on
type root struct {
	t                 *testing.T
	serverConfig      *TestingSSHServerConfig
	chanNotifications chan interface{}
	*memFile
	files     map[string]*memFile
	filesLock sync.Mutex
	mockErr   error
}

// Set a mocked error that the next handler call will return.
// Set to nil to reset for no error.
func (fs *root) returnErr(err error) {
	fs.mockErr = err
}

func (fs *root) fetch(path string) (*memFile, error) {
	if path == "/" {
		return fs.memFile, nil
	}
	if file, ok := fs.files[path]; ok {
		return file, nil
	}
	return nil, os.ErrNotExist
}

// Implements os.FileInfo, Reader and Writer interfaces.
// These are the 3 interfaces necessary for the Handlers.
type memFile struct {
	name        string
	modtime     time.Time
	symlink     string
	isdir       bool
	content     []byte
	contentLock sync.RWMutex
}

// factory to make sure modtime is set
func newMemFile(name string, isdir bool) *memFile {
	return &memFile{
		name:    name,
		modtime: time.Now(),
		isdir:   isdir,
	}
}

// Have memFile fulfill os.FileInfo interface
func (f *memFile) Name() string { return filepath.Base(f.name) }
func (f *memFile) Size() int64  { return int64(len(f.content)) }
func (f *memFile) Mode() os.FileMode {
	ret := os.FileMode(0644)
	if f.isdir {
		ret = os.FileMode(0755) | os.ModeDir
	}
	if f.symlink != "" {
		ret = os.FileMode(0777) | os.ModeSymlink
	}
	return ret
}
func (f *memFile) ModTime() time.Time { return f.modtime }
func (f *memFile) IsDir() bool        { return f.isdir }
func (f *memFile) Sys() interface{} {
	return fakeFileInfoSys()
}

// Read/Write
func (f *memFile) ReaderAt() (io.ReaderAt, error) {
	if f.isdir {
		return nil, os.ErrInvalid
	}
	return bytes.NewReader(f.content), nil
}

func (f *memFile) WriterAt() (io.WriterAt, error) {
	if f.isdir {
		return nil, os.ErrInvalid
	}
	return f, nil
}
func (f *memFile) WriteAt(p []byte, off int64) (int, error) {
	// fmt.Println(string(p), off)
	// mimic write delays, should be optional
	time.Sleep(time.Microsecond * time.Duration(len(p)))
	f.contentLock.Lock()
	defer f.contentLock.Unlock()
	plen := len(p) + int(off)
	if plen >= len(f.content) {
		nc := make([]byte, plen)
		copy(nc, f.content)
		f.content = nc
	}
	copy(f.content[off:], p)
	return len(p), nil
}

func fakeFileInfoSys() interface{} {
	return &syscall.Stat_t{Uid: 65534, Gid: 65534}
}
