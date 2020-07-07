package resolvers

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/db"
	mhelpers "gitlab.com/olaris/olaris-server/metadata/helpers"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultTimeOffset = -24 * time.Hour

var rescanningLibraries bool

// Library wrapper around the db.Library package so it can contain related resolvers.
type Library struct {
	db.Library
	Movies   []*MovieResolver
	Episodes []*EpisodeResolver
}

// LibraryResolver resolver for Library.
type LibraryResolver struct {
	r Library
}

// IsRefreshing tells us whether the library is currently doing a scan.
func (r *LibraryResolver) IsRefreshing() bool {
	if r.r.RefreshCompletedAt.IsZero() {
		return true
	}

	return false
}

// Name returns library name
func (r *LibraryResolver) Name() string {
	return r.r.Name
}

// Healthy returns library name
func (r *LibraryResolver) Healthy() bool {
	return r.r.Healthy
}

// Backend returns library's backend type
func (r *LibraryResolver) Backend() int32 {
	return int32(r.r.Backend)
}

// RcloneName returns library Rclonename
func (r *LibraryResolver) RcloneName() *string {
	return &r.r.RcloneName
}

// ID returns library ID
func (r *LibraryResolver) ID() int32 {
	return int32(r.r.ID)
}

// Movies returns movies in Library.
func (r *LibraryResolver) Movies(ctx context.Context) []*MovieResolver {
	var mr []*MovieResolver
	for _, movie := range db.FindMoviesInLibrary(r.r.ID) {
		if movie.Title != "" {
			mov := MovieResolver{r: movie}
			mr = append(mr, &mov)
		}
	}
	return mr
}

// Series return seasons based on episodes in a Library.
func (r *LibraryResolver) Series() (series []*SeriesResolver) {
	for _, s := range db.FindSeriesInLibrary(r.r.ID) {
		series = append(series, &SeriesResolver{r: s})
	}
	return series
}

// Episodes returns episodes in Library.
func (r *LibraryResolver) Episodes() (eps []*EpisodeResolver) {
	for _, episode := range db.FindEpisodesInLibrary(r.r.ID) {
		eps = append(eps, &EpisodeResolver{r: episode})
	}

	return eps
}

// FilePath returns filesystem path for library.
func (r *LibraryResolver) FilePath() string {
	return r.r.FilePath
}

// Kind returns library type.
func (r *LibraryResolver) Kind() int32 {
	return int32(r.r.Kind)
}

type createLibraryArgs struct {
	Name       string
	FilePath   string
	Kind       int32
	Backend    int32
	RcloneName *string
}

// RefreshAgentMetadata refreshes all metadata from agent
func (r *Resolver) RefreshAgentMetadata(ctx context.Context, args struct {
	LibraryID *int32
	UUID      *string
}) bool {

	err := ifAdmin(ctx)
	if err != nil {
		return false
	}

	if args.LibraryID != nil {
		// TODO(Leon Handreke): Either add a refresh-per-library call to the LibraryManager
		//  or make this a global update call without a Library ID

		libID := uint(*args.LibraryID)

		for _, lm := range r.libs {
			log.Println(lm.Library.ID, libID)
			if lm.Library.ID == libID {
				go mhelpers.WithLock(func() {
					if lm.Library.Kind == db.MediaTypeMovie {
						r.env.MetadataManager.RefreshAllMovieMetadata()
					} else if lm.Library.Kind == db.MediaTypeSeries {
						r.env.MetadataManager.RefreshAllSeriesMetadata()
					}
				}, "libid"+strconv.FormatUint(uint64(lm.Library.ID), 10))
			}
		}
		return true
	}

	if args.UUID != nil {
		return r.env.MetadataManager.RefreshAgentMetadataForUUID(*args.UUID)
	}

	return false
}

// RescanLibrary can scan (parts) of a Library
func (r *Resolver) RescanLibrary(ctx context.Context, args struct {
	ID       *int32
	FilePath *string
}) bool {
	err := ifAdmin(ctx)
	if err != nil {
		return false
	}

	// No specific library is given
	if args.ID == nil {
		if args.FilePath == nil {
			return false
		}

		// A valid filepath has been given so let's look in all libraries for the given path
		validLibFound := false
		for _, man := range r.libs {
			if strings.Contains(*args.FilePath, man.Library.FilePath) {
				validLibFound = true
				go mhelpers.WithLock(func() {
					man.RescanFilesystem(*args.FilePath)
				}, fmt.Sprintf("refresh-lib-%s", strconv.Itoa(int(man.Library.ID))))
			}
		}
		return validLibFound
	}

	// A specific library has been given
	libId := uint(*args.ID)
	man := r.libs[libId]

	// No specific filepath has been given so we can refresh the whole library.
	if args.FilePath == nil {
		go mhelpers.WithLock(func() {
			man.RefreshAll()
		}, fmt.Sprintf("refresh-lib-%s", strconv.Itoa(int(man.Library.ID))))
	} else {
		go mhelpers.WithLock(func() {
			man.RescanFilesystem(*args.FilePath)
		}, fmt.Sprintf("refresh-lib-%s", strconv.Itoa(int(man.Library.ID))))
	}
	return true
}

// RescanLibraries rescans all libraries for new files.
func (r *Resolver) RescanLibraries(ctx context.Context) bool {
	err := ifAdmin(ctx)
	if err != nil {
		return false
	}

	if rescanningLibraries == false {
		rescanningLibraries = true
		go func() {
			for _, lm := range r.libs {
				lm.RefreshAll()
			}
			rescanningLibraries = false
		}()
		return true
	}
	return false
}

// DeleteLibrary deletes a library.
func (r *Resolver) DeleteLibrary(ctx context.Context, args struct{ ID int32 }) *LibResResolv {
	err := ifAdmin(ctx)
	if err != nil {
		return &LibResResolv{LibraryResponse{Error: CreateErrResolver(err)}}
	}

	libraryManager := r.libs[uint(args.ID)]
	library := *libraryManager.Library
	// TODO(Leon Handreke): Ideally, it would be more explicit what is happening here.
	// We are stopping the watcher to then remove the library manager
	libraryManager.Shutdown()
	libraryManager.DeleteLibrary()

	var libRes LibraryResponse
	// TODO(Maran): Dry up resolver creation here and in CreateLibrary
	if err == nil {
		// TODO(Leon Handreke): Why are returning a deleted library?
		libRes = LibraryResponse{Library: &LibraryResolver{
			Library{library, nil, nil}}}
	} else {
		libRes = LibraryResponse{Error: CreateErrResolver(err)}
	}
	return &LibResResolv{libRes}
}

func errResponse(err error) *LibResResolv {
	return &LibResResolv{LibraryResponse{Error: CreateErrResolver(err)}}
}

// CreateLibrary creates a library.
func (r *Resolver) CreateLibrary(ctx context.Context, args *createLibraryArgs) *LibResResolv {
	var library db.Library
	var err error
	var libRes LibraryResponse
	args.FilePath = filepath.Clean(args.FilePath)

	err = ifAdmin(ctx)
	if err != nil {
		return errResponse(err)
	}

	var rcloneName string

	if args.RcloneName != nil {
		rcloneName = *args.RcloneName
	}

	library = db.Library{Name: args.Name, FilePath: args.FilePath, Kind: db.MediaType(args.Kind), Backend: int(args.Backend), RcloneName: rcloneName}

	// Make sure we don't initialize the library with zero time (issue with strict mode in MySQL)
	library.RefreshStartedAt = time.Now().Add(defaultTimeOffset)
	library.RefreshCompletedAt = time.Now().Add(defaultTimeOffset)

	err = db.AddLibrary(&library)

	if err == nil {
		r.AddLibraryManager(&library)
		libRes = LibraryResponse{Library: &LibraryResolver{Library{library, nil, nil}}}
	} else {
		// TODO(Maran): We probably want to not do this in the resolver but in the database layer so that it gets scanned no matter how you add it.
		// libRes = LibraryResponse{Error: CreateErrResolver(err)}
		return errResponse(err)
	}
	return &LibResResolv{libRes}
}

// LibResResolv holds a library response.
type LibResResolv struct {
	r LibraryResponse
}

// Library returns the library.
func (r *LibResResolv) Library() *LibraryResolver {
	return r.r.Library
}

// Error returns an error.
func (r *LibResResolv) Error() *ErrorResolver {
	return r.r.Error
}

// LibraryResponse generic response.
type LibraryResponse struct {
	Error   *ErrorResolver
	Library *LibraryResolver
}

// Libraries return all libraries.
func (r *Resolver) Libraries(ctx context.Context) []*LibraryResolver {
	var l []*LibraryResolver
	libraries := db.AllLibraries()
	for _, library := range libraries {
		list := Library{library, nil, nil}
		lib := LibraryResolver{r: list}
		l = append(l, &lib)
	}
	return l
}
