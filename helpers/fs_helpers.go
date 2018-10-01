package helpers

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
)

// GetHome returns the given users home folder
func GetHome() string {
	usr, err := user.Current()
	if err != nil {
		panic(fmt.Sprintf("Failed to determine user's home directory, error: '%s'\n", err.Error()))
	}
	return usr.HomeDir
}

// EnsurePath ensures the given filesystem path exists, if not it will create it.
func EnsurePath(pathName string) error {
	glog.Infof("Ensuring folder %s exists.\n", pathName)
	if _, err := os.Stat(pathName); os.IsNotExist(err) {
		glog.Infof("Path %s does not exist, creating", pathName)
		err = os.MkdirAll(pathName, 0755)
		if err != nil {
			glog.Errorf("Could not create %s", pathName)
			return err
		}
	}
	return nil
}

// FileExists checks whether a file or folder exists in the filesystem, will follow symlinks and ensures the target exists too.
func FileExists(pathName string) bool {
	fi, err := os.Lstat(pathName)
	if err != nil {
		return false
	}
	// Is a symlink
	if fi.Mode()&os.ModeSymlink != 0 {
		p, err := filepath.EvalSymlinks(pathName)
		if err != nil {
			return FileExists(p)
		}
	}
	return true
}

// BaseConfigPath returns the root for our config folders.
func BaseConfigPath() string {
	return path.Join(GetHome(), ".config", "olaris")
}

// MetadataConfigPath returns the config path for the md server
func MetadataConfigPath() string {
	return path.Join(BaseConfigPath(), "metadb")
}

// CacheDir returns a cache folder to use.
func CacheDir() string {
	cacheDir, err := UserCacheDir()
	if err != nil {
		panic(fmt.Sprintf("Error getting user cache dir: %s", err.Error()))
	}
	return path.Join(cacheDir, "olaris")
}

// LogPath returns the path to our logfolder.
func LogPath() string {
	logPath := path.Join(CacheDir(), "log")
	EnsurePath(logPath)
	return logPath
}

// UserCacheDir returns the default root directory to use for user-specific
// cached data. Users should create their own application-specific subdirectory
// within this one and use that.
//
// On Unix systems, it returns $XDG_CACHE_HOME as specified by
// https://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html if
// non-empty, else $HOME/.cache.
// On Darwin, it returns $HOME/Library/Caches.
// On Windows, it returns %LocalAppData%.
// On Plan 9, it returns $home/lib/cache.
//
// If the location cannot be determined (for example, $HOME is not defined),
// then it will return an error.
func UserCacheDir() (string, error) {
	var dir string

	// TODO(Leon Handreke): This is in os in golang 1.11
	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("LocalAppData")
		if dir == "" {
			return "", errors.New("%LocalAppData% is not defined")
		}

	case "darwin":
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}
		dir += "/Library/Caches"

	case "plan9":
		dir = os.Getenv("home")
		if dir == "" {
			return "", errors.New("$home is not defined")
		}
		dir += "/lib/cache"

	default: // Unix
		dir = os.Getenv("XDG_CACHE_HOME")
		if dir == "" {
			dir = os.Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CACHE_HOME nor $HOME are defined")
			}
			dir += "/.cache"
		}
	}

	return dir, nil
}
