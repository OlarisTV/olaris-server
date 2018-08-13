package resolvers

import (
	"context"
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
)

type Library struct {
	db.Library
	Movies   []*MovieResolver
	Episodes []*EpisodeResolver
}

type LibraryResolver struct {
	r Library
}

func (r *LibraryResolver) Name() string {
	return r.r.Name
}

func (r *LibraryResolver) ID() int32 {
	return int32(r.r.ID)
}

func (r *LibraryResolver) Movies() []*MovieResolver {
	return r.r.Movies
}
func (r *LibraryResolver) Episodes() []*EpisodeResolver {
	return r.r.Episodes
}
func (r *LibraryResolver) FilePath() string {
	return r.r.FilePath
}
func (r *LibraryResolver) Kind() int32 {
	return int32(r.r.Kind)
}

type CreateLibraryArgs struct {
	Name     string
	FilePath string
	Kind     int32
}

func (r *Resolver) DeleteLibrary(args struct{ ID int32 }) *libResResolv {
	library, err := db.DeleteLibrary(int(args.ID))
	var libRes LibRes
	// TODO(Maran): Dry up resolver creation here and in CreateLibrary
	if err == nil {
		libRes = LibRes{Error: &ErrorResolver{Error{hasError: false}}, Library: &LibraryResolver{Library{library, nil, nil}}}
	} else {
		libRes = LibRes{Error: &ErrorResolver{Error{hasError: true, message: err.Error()}}, Library: &LibraryResolver{Library{}}}
	}
	return &libResResolv{libRes}
}

func (r *Resolver) CreateLibrary(args *CreateLibraryArgs) *libResResolv {
	library, err := db.AddLibrary(args.Name, args.FilePath, db.MediaType(args.Kind))
	var libRes LibRes
	if err == nil {
		fmt.Println("Scaninng library")
		// TODO(Maran): We probably want to not do this in the resolver but in the database layer so that it gets scanned no matter how you add it.
		go db.NewLibraryManager(r.env.Watcher).RefreshAll()
		libRes = LibRes{Error: &ErrorResolver{Error{hasError: false}}, Library: &LibraryResolver{Library{library, nil, nil}}}
	} else {
		libRes = LibRes{Error: &ErrorResolver{Error{hasError: true, message: err.Error()}}, Library: &LibraryResolver{Library{}}}
	}
	return &libResResolv{libRes}
}

type libResResolv struct {
	r LibRes
}

func (r *libResResolv) Library() *LibraryResolver {
	return r.r.Library
}
func (r *libResResolv) Error() *ErrorResolver {
	return r.r.Error
}

type LibRes struct {
	Error   *ErrorResolver
	Library *LibraryResolver
}

func (r *Resolver) Libraries(ctx context.Context) []*LibraryResolver {
	userID := GetUserID(ctx)
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
			list.Episodes = append(list.Episodes, &EpisodeResolver{r: episode})
		}

		lib := LibraryResolver{r: list}
		l = append(l, &lib)
	}
	return l
}
