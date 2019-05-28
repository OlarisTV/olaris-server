package managers

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
	// Backends
	_ "github.com/ncw/rclone/backend/drive"
	_ "github.com/ncw/rclone/backend/local"
	"github.com/ncw/rclone/fs"
	"github.com/ncw/rclone/vfs"
	"gitlab.com/olaris/olaris-server/streaming"
	"path/filepath"
	"strings"
)

type FileStat interface {
	Size() int64
	Name() string
	Path() string
	StreamLink() string
	ProbePath() string
	IsDir() bool
}

type RcloneFileStat struct {
	dirEntry   fs.DirEntry
	rcloneName string
}

func (lfs *RcloneFileStat) Name() string {
	return filepath.Base(lfs.dirEntry.String())
}

func NewRcloneFileStatFromFilePath(filePath string) (*RcloneFileStat, error) {
	parts := strings.SplitN(filePath, "/", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("could not detect Rclone data from filePath.\n")
	}
	rcloneName := parts[1]

	filesystem, err := fs.NewFs(rcloneName + ":/")
	if err != nil {
		return nil, err
	}
	fs := vfs.New(filesystem, &vfs.Options{ReadOnly: true, CacheMode: vfs.CacheModeFull})

	node, err := fs.Stat(parts[2])
	if err != nil {
		return nil, err
	}

	return NewRcloneFileStat(node.DirEntry(), rcloneName), nil
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

func (lfs *RcloneFileStat) ProbePath() string {
	probePath := filepath.Join("rclone", lfs.rcloneName, lfs.Path())
	return probePath
}

func (lfs *RcloneFileStat) StreamLink() string {
	streamLink, err := streaming.GetMediaFileURL(lfs.ProbePath())
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
func (lfs *LocalFileStat) ProbePath() string {
	return lfs.Path()
}
func (lfs *LocalFileStat) StreamLink() string {
	return "file://" + lfs.path
}
