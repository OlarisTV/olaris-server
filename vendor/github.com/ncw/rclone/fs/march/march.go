// Package march traverses two directories in lock step
package march

import (
	"context"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/ncw/rclone/fs"
	"github.com/ncw/rclone/fs/filter"
	"github.com/ncw/rclone/fs/list"
	"github.com/ncw/rclone/fs/walk"
	"golang.org/x/text/unicode/norm"
)

// March holds the data used to traverse two Fs simultaneously,
// calling callback for each match
type March struct {
	// parameters
	ctx      context.Context
	fdst     fs.Fs
	fsrc     fs.Fs
	dir      string
	callback Marcher
	// internal state
	srcListDir listDirFn // function to call to list a directory in the src
	dstListDir listDirFn // function to call to list a directory in the dst
	transforms []matchTransformFn
}

// Marcher is called on each match
type Marcher interface {
	// SrcOnly is called for a DirEntry found only in the source
	SrcOnly(src fs.DirEntry) (recurse bool)
	// DstOnly is called for a DirEntry found only in the destination
	DstOnly(dst fs.DirEntry) (recurse bool)
	// Match is called for a DirEntry found both in the source and destination
	Match(dst, src fs.DirEntry) (recurse bool)
}

// New sets up a march over fsrc, and fdst calling back callback for each match
func New(ctx context.Context, fdst, fsrc fs.Fs, dir string, callback Marcher) *March {
	m := &March{
		ctx:      ctx,
		fdst:     fdst,
		fsrc:     fsrc,
		dir:      dir,
		callback: callback,
	}
	m.srcListDir = m.makeListDir(fsrc, false)
	m.dstListDir = m.makeListDir(fdst, filter.Active.Opt.DeleteExcluded)
	// Now create the matching transform
	// ..normalise the UTF8 first
	m.transforms = append(m.transforms, norm.NFC.String)
	// ..if destination is caseInsensitive then make it lower case
	// case Insensitive | src | dst | lower case compare |
	//                  | No  | No  | No                 |
	//                  | Yes | No  | No                 |
	//                  | No  | Yes | Yes                |
	//                  | Yes | Yes | Yes                |
	if fdst.Features().CaseInsensitive {
		m.transforms = append(m.transforms, strings.ToLower)
	}
	return m
}

// list a directory into entries, err
type listDirFn func(dir string) (entries fs.DirEntries, err error)

// makeListDir makes a listing function for the given fs and includeAll flags
func (m *March) makeListDir(f fs.Fs, includeAll bool) listDirFn {
	if !fs.Config.UseListR || f.Features().ListR == nil {
		return func(dir string) (entries fs.DirEntries, err error) {
			return list.DirSorted(f, includeAll, dir)
		}
	}
	var (
		mu      sync.Mutex
		started bool
		dirs    walk.DirTree
		dirsErr error
	)
	return func(dir string) (entries fs.DirEntries, err error) {
		mu.Lock()
		defer mu.Unlock()
		if !started {
			dirs, dirsErr = walk.NewDirTree(f, m.dir, includeAll, fs.Config.MaxDepth)
			started = true
		}
		if dirsErr != nil {
			return nil, dirsErr
		}
		entries, ok := dirs[dir]
		if !ok {
			err = fs.ErrorDirNotFound
		} else {
			delete(dirs, dir)
		}
		return entries, err
	}
}

// listDirJob describe a directory listing that needs to be done
type listDirJob struct {
	srcRemote string
	dstRemote string
	srcDepth  int
	dstDepth  int
	noSrc     bool
	noDst     bool
}

// Run starts the matching process off
func (m *March) Run() {
	srcDepth := fs.Config.MaxDepth
	if srcDepth < 0 {
		srcDepth = fs.MaxLevel
	}
	dstDepth := srcDepth
	if filter.Active.Opt.DeleteExcluded {
		dstDepth = fs.MaxLevel
	}

	// Start some directory listing go routines
	var wg sync.WaitGroup         // sync closing of go routines
	var traversing sync.WaitGroup // running directory traversals
	in := make(chan listDirJob, fs.Config.Checkers)
	for i := 0; i < fs.Config.Checkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-m.ctx.Done():
					return
				case job, ok := <-in:
					if !ok {
						return
					}
					jobs := m.processJob(job)
					if len(jobs) > 0 {
						traversing.Add(len(jobs))
						go func() {
							// Now we have traversed this directory, send these
							// jobs off for traversal in the background
							for _, newJob := range jobs {
								in <- newJob
							}
						}()
					}
					traversing.Done()
				}
			}
		}()
	}

	// Start the process
	traversing.Add(1)
	in <- listDirJob{
		srcRemote: m.dir,
		srcDepth:  srcDepth - 1,
		dstRemote: m.dir,
		dstDepth:  dstDepth - 1,
	}
	traversing.Wait()
	close(in)
	wg.Wait()
}

// Check to see if the context has been cancelled
func (m *March) aborting() bool {
	select {
	case <-m.ctx.Done():
		return true
	default:
	}
	return false
}

// matchEntry is an entry plus transformed name
type matchEntry struct {
	entry fs.DirEntry
	leaf  string
	name  string
}

// matchEntries contains many matchEntry~s
type matchEntries []matchEntry

// Len is part of sort.Interface.
func (es matchEntries) Len() int { return len(es) }

// Swap is part of sort.Interface.
func (es matchEntries) Swap(i, j int) { es[i], es[j] = es[j], es[i] }

// Less is part of sort.Interface.
//
// Compare in order (name, leaf, remote)
func (es matchEntries) Less(i, j int) bool {
	ei, ej := &es[i], &es[j]
	if ei.name == ej.name {
		if ei.leaf == ej.leaf {
			return ei.entry.Remote() < ej.entry.Remote()
		}
		return ei.leaf < ej.leaf
	}
	return ei.name < ej.name
}

// Sort the directory entries by (name, leaf, remote)
//
// We use a stable sort here just in case there are
// duplicates. Assuming the remote delivers the entries in a
// consistent order, this will give the best user experience
// in syncing as it will use the first entry for the sync
// comparison.
func (es matchEntries) sort() {
	sort.Stable(es)
}

// make a matchEntries from a newMatch entries
func newMatchEntries(entries fs.DirEntries, transforms []matchTransformFn) matchEntries {
	es := make(matchEntries, len(entries))
	for i := range es {
		es[i].entry = entries[i]
		name := path.Base(entries[i].Remote())
		es[i].leaf = name
		for _, transform := range transforms {
			name = transform(name)
		}
		es[i].name = name
	}
	es.sort()
	return es
}

// matchPair is a matched pair of direntries returned by matchListings
type matchPair struct {
	src, dst fs.DirEntry
}

// matchTransformFn converts a name into a form which is used for
// comparison in matchListings.
type matchTransformFn func(name string) string

// Process the two listings, matching up the items in the two slices
// using the transform function on each name first.
//
// Into srcOnly go Entries which only exist in the srcList
// Into dstOnly go Entries which only exist in the dstList
// Into matches go matchPair's of src and dst which have the same name
//
// This checks for duplicates and checks the list is sorted.
func matchListings(srcListEntries, dstListEntries fs.DirEntries, transforms []matchTransformFn) (srcOnly fs.DirEntries, dstOnly fs.DirEntries, matches []matchPair) {
	srcList := newMatchEntries(srcListEntries, transforms)
	dstList := newMatchEntries(dstListEntries, transforms)
	for iSrc, iDst := 0, 0; ; iSrc, iDst = iSrc+1, iDst+1 {
		var src, dst fs.DirEntry
		var srcName, dstName string
		if iSrc < len(srcList) {
			src = srcList[iSrc].entry
			srcName = srcList[iSrc].name
		}
		if iDst < len(dstList) {
			dst = dstList[iDst].entry
			dstName = dstList[iDst].name
		}
		if src == nil && dst == nil {
			break
		}
		if src != nil && iSrc > 0 {
			prev := srcList[iSrc-1].name
			if srcName == prev {
				fs.Logf(src, "Duplicate %s found in source - ignoring", fs.DirEntryType(src))
				iDst-- // ignore the src and retry the dst
				continue
			} else if srcName < prev {
				// this should never happen since we sort the listings
				panic("Out of order listing in source")
			}
		}
		if dst != nil && iDst > 0 {
			prev := dstList[iDst-1].name
			if dstName == prev {
				fs.Logf(dst, "Duplicate %s found in destination - ignoring", fs.DirEntryType(dst))
				iSrc-- // ignore the dst and retry the src
				continue
			} else if dstName < prev {
				// this should never happen since we sort the listings
				panic("Out of order listing in destination")
			}
		}
		if src != nil && dst != nil {
			if srcName < dstName {
				dst = nil
				iDst-- // retry the dst
			} else if srcName > dstName {
				src = nil
				iSrc-- // retry the src
			}
		}
		// Debugf(nil, "src = %v, dst = %v", src, dst)
		switch {
		case src == nil && dst == nil:
			// do nothing
		case src == nil:
			dstOnly = append(dstOnly, dst)
		case dst == nil:
			srcOnly = append(srcOnly, src)
		default:
			matches = append(matches, matchPair{src: src, dst: dst})
		}
	}
	return
}

// processJob processes a listDirJob listing the source and
// destination directories, comparing them and returning a slice of
// more jobs
//
// returns errors using processError
func (m *March) processJob(job listDirJob) (jobs []listDirJob) {
	var (
		srcList, dstList       fs.DirEntries
		srcListErr, dstListErr error
		wg                     sync.WaitGroup
	)

	// List the src and dst directories
	if !job.noSrc {
		wg.Add(1)
		go func() {
			defer wg.Done()
			srcList, srcListErr = m.srcListDir(job.srcRemote)
		}()
	}
	if !job.noDst {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dstList, dstListErr = m.dstListDir(job.dstRemote)
		}()
	}

	// Wait for listings to complete and report errors
	wg.Wait()
	if srcListErr != nil {
		fs.Errorf(job.srcRemote, "error reading source directory: %v", srcListErr)
		fs.CountError(srcListErr)
		return nil
	}
	if dstListErr == fs.ErrorDirNotFound {
		// Copy the stuff anyway
	} else if dstListErr != nil {
		fs.Errorf(job.dstRemote, "error reading destination directory: %v", dstListErr)
		fs.CountError(dstListErr)
		return nil
	}

	// Work out what to do and do it
	srcOnly, dstOnly, matches := matchListings(srcList, dstList, m.transforms)
	for _, src := range srcOnly {
		if m.aborting() {
			return nil
		}
		recurse := m.callback.SrcOnly(src)
		if recurse && job.srcDepth > 0 {
			jobs = append(jobs, listDirJob{
				srcRemote: src.Remote(),
				srcDepth:  job.srcDepth - 1,
				noDst:     true,
			})
		}

	}
	for _, dst := range dstOnly {
		if m.aborting() {
			return nil
		}
		recurse := m.callback.DstOnly(dst)
		if recurse && job.dstDepth > 0 {
			jobs = append(jobs, listDirJob{
				dstRemote: dst.Remote(),
				dstDepth:  job.dstDepth - 1,
				noSrc:     true,
			})
		}
	}
	for _, match := range matches {
		if m.aborting() {
			return nil
		}
		recurse := m.callback.Match(match.dst, match.src)
		if recurse && job.srcDepth > 0 && job.dstDepth > 0 {
			jobs = append(jobs, listDirJob{
				srcRemote: match.src.Remote(),
				dstRemote: match.dst.Remote(),
				srcDepth:  job.srcDepth - 1,
				dstDepth:  job.dstDepth - 1,
			})
		}
	}
	return jobs
}
