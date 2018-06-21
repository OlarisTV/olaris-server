package resolvers

import (
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
)

type UuidArgs struct {
	Uuid *string
}

func (r *Resolver) Movies(args *UuidArgs) []*MovieResolver {
	var l []*MovieResolver
	var movies []db.MovieItem
	if args.Uuid != nil {
		movies = db.FindMovieWithUUID(args.Uuid)
	} else {
		movies = db.FindAllMovies()
	}
	for _, movie := range movies {
		if movie.Title != "" {
			mov := MovieResolver{r: movie}
			l = append(l, &mov)
		}
	}
	return l
}

type MovieResolver struct {
	r db.MovieItem
}

func (r *MovieResolver) Title() string {
	return r.r.Title
}
func (r *MovieResolver) UUID() string {
	return r.r.UUID
}
func (r *MovieResolver) OriginalTitle() string {
	return r.r.OriginalTitle
}
func (r *MovieResolver) FilePath() string {
	return r.r.FilePath
}
func (r *MovieResolver) FileName() string {
	return r.r.FileName
}
func (r *MovieResolver) BackdropPath() string {
	return r.r.BackdropPath
}
func (r *MovieResolver) PosterPath() string {
	return r.r.PosterPath
}
func (r *MovieResolver) Year() string {
	return r.r.YearAsString()
}
func (r *MovieResolver) Overview() string {
	return r.r.Overview
}
func (r *MovieResolver) ImdbID() string {
	return r.r.ImdbID
}
func (r *MovieResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}

// Will this be a problem if we ever run out of the 32int space?
func (r *MovieResolver) LibraryID() int32 {
	return int32(r.r.LibraryID)
}
