package resolvers

import (
	"context"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/db"
	mhelpers "gitlab.com/olaris/olaris-server/metadata/helpers"
	"path/filepath"
	"strconv"
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
func (r *Resolver) RefreshAgentMetadata(args struct {
	LibraryID *int32
	UUID      *string
}) bool {
	if args.LibraryID != nil {
		// TODO(Leon Handreke): Either add a refresh-per-library call to the LibraryManager
		//  or make this a global update call without a Library ID

		libID := uint(*args.LibraryID)

		for _, lm := range r.libs {
			log.Println(lm.Library.ID, libID)
			if lm.Library.ID == libID {
				go mhelpers.WithLock(func() {
					if lm.Library.Kind == db.MediaTypeMovie {
						r.env.MetadataManager.ForceMovieMetadataUpdate()
					} else if lm.Library.Kind == db.MediaTypeSeries {
						r.env.MetadataManager.ForceSeriesMetadataUpdate()
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

// RescanLibraries rescans all libraries for new files.
func (r *Resolver) RescanLibraries() bool {
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

	r.StopLibraryManager(uint(args.ID))

	library, err := db.DeleteLibrary(int(args.ID))
	var libRes LibraryResponse
	// TODO(Maran): Dry up resolver creation here and in CreateLibrary
	if err == nil {
		libRes = LibraryResponse{Library: &LibraryResolver{Library{library, nil, nil}}}
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
