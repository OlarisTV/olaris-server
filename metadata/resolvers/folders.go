package resolvers

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/filesystem"
)

type folderArgs struct {
	Path     string
	FullPath bool
}

// Remotes returns all Folders in the given path, takes a FileLocator and returns all folders in the given folder
func (r *Resolver) Folders(ctx context.Context, args *folderArgs) (folders []*string) {
	err := ifAdmin(ctx)
	if err != nil {
		log.Error("unauthorized:", err)
		return folders
	}

	folders, err = findNode(args.Path, args.FullPath, "")
	if err != nil {
		log.Warn("no valid path found for", args.Path, err)
	}
	return folders
}

func findNode(path string, fullPath bool, filename string) (folders []*string, err error) {
	locator, err := filesystem.ParseFileLocator(path)
	if err != nil {
		log.Errorf("%s is not a valid path", path)
		return folders, err
	}

	n, err := filesystem.GetNodeFromFileLocator(locator)

	if err != nil {
		// an error has been thrown which means the given path is not a valid path
		// before giving up we are going to attempt a fuzzy search by splitting getting the latest full folder
		// and then splitting that from the rest and using the last bit as a filter
		// ie. /home/animazing/Do is the given path, 'Do' is not a valid folder but '/home/animazing/' is so the findNode method will be called again
		// Giving '/home/animazing/' as path and 'Do' is (partial) filename. It should now result in a list including the 'Download' folder which lives
		// in '/home/animazing/'
		b := filepath.Dir(path)
		fileName := strings.Split(path, b)
		if b != "" && len(fileName) > 0 {
			log.Debugln("couldn't find full path, going to attempt a fuzzy search")
			locator, err := filesystem.ParseFileLocator(b)
			if err != nil {
				log.Errorf("%s is not a valid path", path)
				return folders, err
			}

			_, err = filesystem.GetNodeFromFileLocator(locator)
			if err == nil {
				log.Debugf("Found a valid path: '%s', filtering by filename '%s' now", b, fileName[1][1:])
				return findNode(b, fullPath, fileName[1][1:])
			} else {
				return folders, err
			}
		}
	}

	f, err := n.ListDir()

	if err != nil {
		return folders, err
	}

	sort.Strings(f)

	for i := range f {
		if filename != "" {
			if !(strings.Contains(f[i], filename)) {
				continue
			}
		}

		if fullPath {
			f[i] = filepath.Join(locator.Path, f[i])
		}

		folders = append(folders, &f[i])

	}

	return folders, err
}
