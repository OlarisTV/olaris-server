package resolvers

import (
	"context"
	"fmt"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers"
)

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

// Name returns library name
func (r *LibraryResolver) Name() string {
	return r.r.Name
}

// ID returns library ID
func (r *LibraryResolver) ID() int32 {
	return int32(r.r.ID)
}

// Movies returns movies in Library.
func (r *LibraryResolver) Movies() []*MovieResolver {
	return r.r.Movies
}

// Episodes returns episodes in Library.
func (r *LibraryResolver) Episodes() []*EpisodeResolver {
	return r.r.Episodes
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
	Name     string
	FilePath string
	Kind     int32
}

// DeleteLibrary deletes a library.
func (r *Resolver) DeleteLibrary(ctx context.Context, args struct{ ID int32 }) *LibResResolv {
	err := ifAdmin(ctx)
	if err != nil {
		return &LibResResolv{LibraryResponse{Error: CreateErrResolver(err)}}
	}

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

// CreateLibrary creates a library.
func (r *Resolver) CreateLibrary(ctx context.Context, args *createLibraryArgs) *LibResResolv {
	var library db.Library
	var err error
	var libRes LibraryResponse

	err = ifAdmin(ctx)
	if err != nil {
		return &LibResResolv{LibraryResponse{Error: CreateErrResolver(err)}}
	}

	if err == nil {
		library, err = db.AddLibrary(args.Name, args.FilePath, db.MediaType(args.Kind))
		fmt.Println("Scanning library")
		// TODO(Maran): We probably want to not do this in the resolver but in the database layer so that it gets scanned no matter how you add it.
		go managers.NewLibraryManager(r.env.Watcher).RefreshAll()
		libRes = LibraryResponse{Library: &LibraryResolver{Library{library, nil, nil}}}
	} else {
		libRes = LibraryResponse{Error: CreateErrResolver(err)}
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
	userID, _ := auth.UserID(ctx)
	var l []*LibraryResolver
	libraries := db.AllLibraries()
	for _, library := range libraries {
		list := Library{library, nil, nil}
		var mr []*MovieResolver
		for _, movie := range db.FindMoviesInLibrary(library.ID, userID) {
			if movie.Title != "" {
				mov := MovieResolver{r: movie}
				mr = append(mr, &mov)
			}
		}
		list.Movies = mr

		for _, episode := range db.FindEpisodesInLibrary(library.ID, userID) {
			list.Episodes = append(list.Episodes, &EpisodeResolver{r: newEpisode(&episode, userID)})
		}

		lib := LibraryResolver{r: list}
		l = append(l, &lib)
	}
	return l
}
