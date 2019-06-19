package filesystem

import (
	"fmt"
	"path"
	"strings"
)

// BackendType specifies what kind of Library backend is being used.
type BackendType int

const (
	// BackendLocal is used for local libraries
	BackendLocal = iota
	// BackendRclone is used for Rclone remotes
	BackendRclone
)

type FileLocator struct {
	Backend BackendType
	Path    string
}

var backendTypeToString = map[BackendType]string{
	BackendLocal:  "local",
	BackendRclone: "rclone",
}

func (fl FileLocator) String() string {
	return fmt.Sprintf("%s#%s", backendTypeToString[fl.Backend], fl.Path)
}

type WalkFunc func(path string, node Node, err error) error

type Node interface {
	BackendType() BackendType
	Size() int64
	Name() string
	Path() string
	IsDir() bool
	Walk(walkFunc WalkFunc, followFileSymlinks bool) error
	FileLocator() FileLocator
}

func ParseFileLocator(locatorStr string) (FileLocator, error) {
	if locatorStr[0] == '/' {
		locatorStr = locatorStr[1:]

	}
	parts := strings.SplitN(locatorStr, "#", 2)

	if len(parts) != 2 {
		return FileLocator{}, fmt.Errorf("\"%s\" is not a file locator string", locatorStr)
	}
	if parts[0] == "rclone" {
		return FileLocator{BackendRclone, parts[1]}, nil
	} else if parts[0] == "local" {
		return FileLocator{BackendLocal, parts[1]}, nil
	}
	// Don't require an explicit local prefix for now
	return FileLocator{BackendLocal, path.Clean("/" + locatorStr)}, nil
}

func GetNodeFromFileLocator(l FileLocator) (Node, error) {
	if l.Backend == BackendLocal {
		return LocalNodeFromPath(l.Path)
	} else if l.Backend == BackendRclone {
		return RcloneNodeFromPath(l.Path)

	}
	return nil, fmt.Errorf("No such backend: %d", l.Backend)
}
