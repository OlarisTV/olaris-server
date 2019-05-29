package filesystem

import (
	"fmt"
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

type fileLocator struct {
	Backend BackendType
	Path    string
}

type WalkFunc func(path string, node Node, err error) error

type Node interface {
	BackendType() BackendType
	Size() int64
	Name() string
	Path() string
	FfmpegUrl() string
	IsDir() bool
	Walk(walkFunc WalkFunc) error
	FileLocator() string
}

func splitLocatorString(locatorStr string) (fileLocator, error) {
	if locatorStr[0] == '/' {
		locatorStr = locatorStr[1:]

	}
	parts := strings.SplitN(locatorStr, "/", 2)

	if len(parts) != 2 {
		return fileLocator{}, fmt.Errorf("\"%s\" is not a file locator string", locatorStr)
	}
	if parts[0] == "rclone" {
		return fileLocator{BackendRclone, "/" + parts[1]}, nil
	} else if parts[0] == "local" {
		return fileLocator{BackendLocal, "/" + parts[1]}, nil
	}
	// Don't require an explicit local prefix for now
	return fileLocator{BackendLocal, "/" + locatorStr}, nil
}

func GetNode(locator string) (Node, error) {
	l, err := splitLocatorString(locator)
	if err != nil {
		return nil, err
	}

	if l.Backend == BackendLocal {
		return LocalNodeFromPath(l.Path)
	} else if l.Backend == BackendRclone {
		return RcloneNodeFromPath(l.Path)

	}
	return nil, fmt.Errorf("No such backend: %d", l.Backend)
}
