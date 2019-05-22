// Package list contains list functions
package list

import (
	"sort"
	"strings"

	"github.com/ncw/rclone/fs"
	"github.com/ncw/rclone/fs/filter"
	"github.com/pkg/errors"
)

// DirSorted reads Object and *Dir into entries for the given Fs.
//
// dir is the start directory, "" for root
//
// If includeAll is specified all files will be added, otherwise only
// files and directories passing the filter will be added.
//
// Files will be returned in sorted order
func DirSorted(f fs.Fs, includeAll bool, dir string) (entries fs.DirEntries, err error) {
	// Get unfiltered entries from the fs
	entries, err = f.List(dir)
	if err != nil {
		return nil, err
	}
	// This should happen only if exclude files lives in the
	// starting directory, otherwise ListDirSorted should not be
	// called.
	if !includeAll && filter.Active.ListContainsExcludeFile(entries) {
		fs.Debugf(dir, "Excluded")
		return nil, nil
	}
	return filterAndSortDir(entries, includeAll, dir, filter.Active.IncludeObject, filter.Active.IncludeDirectory(f))
}

// filter (if required) and check the entries, then sort them
func filterAndSortDir(entries fs.DirEntries, includeAll bool, dir string,
	IncludeObject func(o fs.Object) bool,
	IncludeDirectory func(remote string) (bool, error)) (newEntries fs.DirEntries, err error) {
	newEntries = entries[:0] // in place filter
	prefix := ""
	if dir != "" {
		prefix = dir + "/"
	}
	for _, entry := range entries {
		ok := true
		// check includes and types
		switch x := entry.(type) {
		case fs.Object:
			// Make sure we don't delete excluded files if not required
			if !includeAll && !IncludeObject(x) {
				ok = false
				fs.Debugf(x, "Excluded")
			}
		case fs.Directory:
			if !includeAll {
				include, err := IncludeDirectory(x.Remote())
				if err != nil {
					return nil, err
				}
				if !include {
					ok = false
					fs.Debugf(x, "Excluded")
				}
			}
		default:
			return nil, errors.Errorf("unknown object type %T", entry)
		}
		// check remote name belongs in this directry
		remote := entry.Remote()
		switch {
		case !ok:
			// ignore
		case !strings.HasPrefix(remote, prefix):
			ok = false
			fs.Errorf(entry, "Entry doesn't belong in directory %q (too short) - ignoring", dir)
		case remote == prefix:
			ok = false
			fs.Errorf(entry, "Entry doesn't belong in directory %q (same as directory) - ignoring", dir)
		case strings.ContainsRune(remote[len(prefix):], '/'):
			ok = false
			fs.Errorf(entry, "Entry doesn't belong in directory %q (contains subdir) - ignoring", dir)
		default:
			// ok
		}
		if ok {
			newEntries = append(newEntries, entry)
		}
	}
	entries = newEntries

	// Sort the directory entries by Remote
	//
	// We use a stable sort here just in case there are
	// duplicates. Assuming the remote delivers the entries in a
	// consistent order, this will give the best user experience
	// in syncing as it will use the first entry for the sync
	// comparison.
	sort.Stable(entries)
	return entries, nil
}
