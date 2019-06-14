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
		return nil, err
	}
	return &LocalNode{fileInfo: fileInfo, path: path}, nil
}

func (n *LocalNode) Name() string {
	return n.fileInfo.Name()
}
func (n *LocalNode) Size() int64 {
	// It appears that when you walk a filepath for nodes it stats the symlink and not the file it links to
	// We do this hack here to ensure we return the size of the actual file, not the symlink.
	if n.fileInfo.Mode()&os.ModeSymlink != 0 {
		p, err := filepath.EvalSymlinks(n.path)
		if err != nil {
			return 0
		}

		fi, err := os.Stat(p)
		if err != nil {
			return 0
		}

		return fi.Size()
	}

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
func (n *LocalNode) Walk(walkFn WalkFunc) error {
	return filepath.Walk(n.path, func(walkPath string, info os.FileInfo, err error) error {
		return walkFn(walkPath, &LocalNode{info, walkPath}, err)
	})
}
