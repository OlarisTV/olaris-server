package resolvers

import (
	"context"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
)

type UuidArgs struct {
	Uuid *string
}
type MustUuidArgs struct {
	Uuid string
}

func (r *Resolver) Movies(ctx context.Context, args *UuidArgs) []*MovieResolver {
	userID := helpers.GetUserID(ctx)
	var l []*MovieResolver
	var movies []db.Movie
	if args.Uuid != nil {
		movies = db.FindMovieWithUUID(args.Uuid, userID)
	} else {
		movies = db.FindAllMovies(userID)
	}
	for _, movie := range movies {
		mov := MovieResolver{r: movie}
		l = append(l, &mov)
	}
	return l
}

type MovieResolver struct {
	r db.Movie
}

func (r *MovieResolver) Files() (res []*MovieFileResolver) {
	for _, file := range r.r.MovieFiles {
		resolver := MovieFileResolver{r: file}
		res = append(res, &resolver)
	}
	return res
}

func (r *MovieResolver) Title() string {
	return r.r.Title
}
func (r *MovieResolver) UUID() string {
	return r.r.UUID
}
func (r *MovieResolver) Name() string {
	return r.r.OriginalTitle
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
func (r *MovieResolver) PlayState() *PlayStateResolver {
	return &PlayStateResolver{r: r.r.PlayState}
}

type MovieFileResolver struct {
	r db.MovieFile
}

// Will this be a problem if we ever run out of the 32int space?
func (r *MovieFileResolver) LibraryID() int32 {
	return int32(r.r.LibraryID)
}
func (r *MovieFileResolver) FilePath() string {
	return r.r.FilePath
}
func (r *MovieFileResolver) FileName() string {
	return r.r.FileName
}
func (r *MovieFileResolver) UUID() string {
	return r.r.UUID
}
func (r *MovieFileResolver) Streams() (streams []*StreamResolver) {
	for _, stream := range r.r.Streams {
		streams = append(streams, &StreamResolver{stream})
	}
	return streams
}
