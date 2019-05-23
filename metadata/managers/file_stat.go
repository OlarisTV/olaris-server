package managers

import (
	"os"
	// Backends
	_ "github.com/ncw/rclone/backend/drive"
	_ "github.com/ncw/rclone/backend/local"
	"github.com/ncw/rclone/fs"
	"path/filepath"
)

type FileStat interface {
	Size() int64
	Name() string
	Path() string
	IsDir() bool
}

type RcloneFileStat struct {
	dirEntry fs.DirEntry
}

func (lfs *RcloneFileStat) Name() string {
	return filepath.Base(lfs.dirEntry.String())
}

func (lfs *RcloneFileStat) Path() string {
	return lfs.dirEntry.String()
}
func (lfs *RcloneFileStat) Size() int64 {
	return lfs.dirEntry.Size()
}
func (lfs *RcloneFileStat) IsDir() bool {
	if lfs.dirEntry.Size() == -1 {
		return true
	} else {
		return false
	}
}

type LocalFileStat struct {
	fileInfo os.FileInfo
	path     string
}

func NewLocalFileStat(filePath string) (*LocalFileStat, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	return &LocalFileStat{fileInfo: fileInfo, path: filePath}, nil
}

func (lfs *LocalFileStat) Name() string {
	return lfs.fileInfo.Name()
}
func (lfs *LocalFileStat) Size() int64 {
	return lfs.fileInfo.Size()
}
func (lfs *LocalFileStat) IsDir() bool {
	return lfs.fileInfo.IsDir()
}
func (lfs *LocalFileStat) Path() string {
	return lfs.path
}
