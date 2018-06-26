package greenskeeper

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

const defaultDirectoryMode = os.FileMode(0701)

type File struct {
	Source string
	Dest   string
	UID    int
	GID    int
	Remove bool
}

func CopyFiles(files []File) error {
	for _, file := range files {
		tmpDest := fmt.Sprintf("%s.tmp", file.Dest)
		fileInfo, err := os.Stat(file.Source)
		if err != nil {
			return err
		}

		src, err := os.Open(file.Source)
		if err != nil {
			return nil
		}
		defer src.Close()

		dst, err := os.Create(tmpDest)
		if err != nil {
			return nil
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}

		if err := os.Chmod(tmpDest, fileInfo.Mode()); err != nil {
			return err
		}

		if file.GID > -1 {
			if err := os.Chown(tmpDest, file.UID, file.GID); err != nil {
				return err
			}
		}

		if err := os.RemoveAll(file.Dest); err != nil {
			return err
		}

		if err := os.Rename(tmpDest, file.Dest); err != nil {
			return err
		}
	}

	return nil
}

type Directory struct {
	Path string
	Mode *os.FileMode
	UID  int
	GID  int

	mkdirAll func(string, os.FileMode) error
	chown    func(string, int, int) error
	chmod    func(string, os.FileMode) error
}

type DirectoryBuilder struct {
	directory Directory
}

func NewDirectoryBuilder(path string) DirectoryBuilder {
	return DirectoryBuilder{directory: Directory{
		Path: path,
		UID:  -1,
		GID:  -1,

		mkdirAll: os.MkdirAll,
		chown:    os.Chown,
		chmod:    os.Chmod,
	}}
}

func (b DirectoryBuilder) Build() Directory {
	return b.directory
}

func (b DirectoryBuilder) UID(uid int) DirectoryBuilder {
	b.directory.UID = uid
	return b
}

func (b DirectoryBuilder) GID(gid int) DirectoryBuilder {
	b.directory.GID = gid
	return b
}

func (b DirectoryBuilder) Mode(mode os.FileMode) DirectoryBuilder {
	b.directory.Mode = &mode
	return b
}

func CreateDirectories(directories ...Directory) error {
	for _, directory := range directories {
		if err := directory.Create(); err != nil {
			return err
		}
	}

	return nil
}

func (d Directory) Create() error {
	if err := d.mkdirAll(d.Path, defaultDirectoryMode); err != nil {
		return err
	}

	if d.Mode != nil {
		if err := d.chmod(d.Path, *d.Mode); err != nil {
			return err
		}
	}

	if d.GID > -1 {
		return d.chown(d.Path, d.UID, d.GID)
	}

	return nil
}

func CheckExistingGdnProcess(pidFilePath string) error {
	return checkExistingGdnProcess(pidFilePath, os.Remove)
}

func checkExistingGdnProcess(pidFilePath string, remove func(string) error) error {
	contents, err := ioutil.ReadFile(pidFilePath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	pid := strings.TrimSpace(string(contents))
	if isRunning(pid) {
		return fmt.Errorf("garden is already running (pid: %s)", pid)
	}

	fmt.Println("Removing stale pidfile...")
	return remove(pidFilePath)
}

func isRunning(pid string) bool {
	if _, err := os.Stat(path.Join("/proc", pid)); pid != "" && err == nil {
		return true
	}
	return false
}

func newFileMode(mode os.FileMode) *os.FileMode {
	fileMode := mode
	return &fileMode
}
