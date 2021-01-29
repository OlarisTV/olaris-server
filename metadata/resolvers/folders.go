package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/filesystem"
	"sort"
)

type folderArgs struct {
	Path string
}

// Remotes returns all Folders in the given path, takes a FileLocator and returns all folders in the given folder
func (r *Resolver) Folders(ctx context.Context, args *folderArgs) (folders []*string) {
	err := ifAdmin(ctx)
	if err != nil {
		return folders
	}

	locator, err := filesystem.ParseFileLocator(args.Path)
	if err != nil {
		return folders
	}

	n, err := filesystem.GetNodeFromFileLocator(locator)
	if err != nil {
		return folders
	}

	f, err := n.ListDir()
	if err != nil {
		return folders
	}

	sort.Strings(f)

	for i := range f {
		folders = append(folders, &f[i])
	}

	return folders
}
