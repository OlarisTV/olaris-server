package filesystem

import (
	"os"
	"path/filepath"
)

type LocalNode struct {
	fileInfo os.FileInfo
	path     string
}

func LocalNodeFromPath(path string) (*LocalNode, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return &LocalNode{path: path}, err
	}
	return &LocalNode{fileInfo: fileInfo, path: path}, nil
}

func (n *LocalNode) Name() string {
	return n.fileInfo.Name()
}
func (n *LocalNode) Size() int64 {
	return n.fileInfo.Size()
}
func (n *LocalNode) IsDir() bool {
	return n.fileInfo.IsDir()
}
func (n *LocalNode) Path() string {
	return n.path
}
func (n *LocalNode) BackendType() BackendType {
	return BackendLocal
}
func (n *LocalNode) FileLocator() FileLocator {
	return FileLocator{Backend: n.BackendType(), Path: n.path}
}
func (n *LocalNode) Walk(walkFn WalkFunc, followFileSymlinks bool) error {
	return filepath.Walk(n.path, func(walkPath string, info os.FileInfo, err error) error {
		// NOTE(Leon Handreke): This behaviour breaks with what filepath.Walk usually does
		// and is a bit weird in general. We should figure out a good strategy here and how
		// to properly abstract it out
		if followFileSymlinks && info.Mode()&os.ModeSymlink != 0 {
			statInfo, err := os.Stat(walkPath)
			if err == nil {
				if !statInfo.IsDir() {
					return walkFn(walkPath, &LocalNode{statInfo, walkPath}, err)
				}
			}
		}
		return walkFn(walkPath, &LocalNode{info, walkPath}, err)
	})
}
