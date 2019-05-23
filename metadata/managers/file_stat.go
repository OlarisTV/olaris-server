package managers

import (
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
	// Backends
	_ "github.com/ncw/rclone/backend/drive"
	_ "github.com/ncw/rclone/backend/local"
	"github.com/ncw/rclone/fs"
	"gitlab.com/olaris/olaris-server/streaming"
	"path/filepath"
	"strings"
)

type FileStat interface {
	Size() int64
	Name() string
	Path() string
	StreamLink() string
	IsDir() bool
}

type RcloneFileStat struct {
	dirEntry   fs.DirEntry
	rcloneName string
}

func (lfs *RcloneFileStat) Name() string {
	return filepath.Base(lfs.dirEntry.String())
}

func NewRcloneFileStat(de fs.DirEntry, name string) *RcloneFileStat {
	return &RcloneFileStat{dirEntry: de, rcloneName: name}
}

func (lfs *RcloneFileStat) Path() string {
	return lfs.dirEntry.String()
}
func (lfs *RcloneFileStat) Size() int64 {
	return lfs.dirEntry.Size()
}
func (lfs *RcloneFileStat) StreamLink() string {
	probePath := filepath.Join("rclone", lfs.rcloneName, lfs.Path())
	streamLink, err := streaming.GetMediaFileURL(probePath)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warnln("Could not get the MediaFileUrl")
	}
	streamLink = strings.Replace(streamLink, lfs.Name(), url.PathEscape(lfs.Name()), -1)
	return streamLink
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
func (lfs *LocalFileStat) StreamLink() string {
	return "file://" + lfs.path
}
