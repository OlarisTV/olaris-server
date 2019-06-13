package filesystem

import (
	"fmt"
	"github.com/ncw/rclone/fs"
	"github.com/ncw/rclone/vfs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"path"
	"strings"
)

type rclonePath struct {
	remoteName string
	path       string
}

func splitRclonePath(pathStr string) (rclonePath, error) {
	if pathStr[0] == '/' {
		pathStr = pathStr[1:]
	}

	parts := strings.SplitN(pathStr, "/", 2)

	if len(parts) != 2 {
		return rclonePath{}, fmt.Errorf("\"%s\" is not an rclone path string", pathStr)
	}

	return rclonePath{remoteName: parts[0], path: parts[1]}, nil
}

type RcloneNode struct {
	// TODO(Leon Handreke): Do we need more abstraction to not make this public?
	// For now, having special handling in the one place where we open the file makes sense.
	Node vfs.Node
}

var vfsCache = map[string]*vfs.VFS{}

func RcloneNodeFromPath(pathStr string) (*RcloneNode, error) {
	l, err := splitRclonePath(pathStr)
	if err != nil {
		return nil, err
	}

	if _, inCache := vfsCache[l.remoteName]; !inCache {
		log.WithFields(log.Fields{"remoteName": l.remoteName}).Debugln("Creating Rclone VFS")
		filesystem, err := fs.NewFs(l.remoteName + ":/")
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create rclone Fs")
		}

		vfsCache[l.remoteName] = vfs.New(filesystem, &vfs.Options{ReadOnly: true,
			CacheMode: vfs.CacheModeFull})
	}
	p := "/" + l.path
	log.WithFields(log.Fields{"path": p, "remoteName": l.remoteName}).Debugln("Checking if Rclone path exists")
	node, err := vfsCache[l.remoteName].Stat(p)
	if err != nil {
		return nil, err
	}
	return &RcloneNode{node}, nil
}

func (n *RcloneNode) Name() string {
	return n.Node.Name()
}

func (n *RcloneNode) Path() string {
	return n.Node.Path()
}
func (n *RcloneNode) Size() int64 {
	return n.Node.Size()
}

func (n *RcloneNode) IsDir() bool {
	return n.Node.IsDir()
}

func (n *RcloneNode) BackendType() BackendType {
	return BackendRclone
}
func (n *RcloneNode) FileLocator() FileLocator {
	// This is a bit of a hack because it seems to be impossible to get the
	// rclone remote name from vfs.Node
	for name, v := range vfsCache {
		if v == n.Node.VFS() {
			return FileLocator{
				Backend: n.BackendType(),
				Path:    path.Join("/", name, n.Path()),
			}
		}
	}
	panic("VFS for given Node not found in cache")
}

func (n *RcloneNode) Walk(walkFn WalkFunc) error {
	if n.Node.IsDir() {
		return walk(n.Node.(*vfs.Dir), walkFn)
	} else {
		return walkFn(n.Path(), n, nil)
	}
}

func walk(root *vfs.Dir, walkFn WalkFunc) error {
	entries, err := root.ReadDirAll()
	if err != nil {
		err = walkFn(root.Path(), &RcloneNode{root}, err)
		if err != nil {
			return err
		}
	} else {
		for _, n := range entries {
			if n.IsDir() {
				err = walk(n.(*vfs.Dir), walkFn)
			} else {
				err = walkFn(n.Path(), &RcloneNode{n}, nil)
			}

			if err != nil {
				return err
			}
		}
	}
	return nil
}
