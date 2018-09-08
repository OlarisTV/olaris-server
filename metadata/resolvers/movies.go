package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

type uuidArgs struct {
	Uuid *string
}

// Movies returns all movies.
func (r *Resolver) Movies(ctx context.Context, args *uuidArgs) []*MovieResolver {
	userID, _ := auth.UserID(ctx)
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

// MovieResolver is a resolver for movies.
type MovieResolver struct {
	r db.Movie
}

// Files return files for movie.
func (r *MovieResolver) Files() (res []*MovieFileResolver) {
	for _, file := range r.r.MovieFiles {
		resolver := MovieFileResolver{r: file}
		res = append(res, &resolver)
	}
	return res
}

// Title returns movie title
func (r *MovieResolver) Title() string {
	return r.r.Title
}

// UUID returns movie uuid
func (r *MovieResolver) UUID() string {
	return r.r.UUID
}

// Name returns movie name
func (r *MovieResolver) Name() string {
	return r.r.OriginalTitle
}

// BackdropPath returns backdrop
func (r *MovieResolver) BackdropPath() string {
	return r.r.BackdropPath
}

// PosterPath returns poster
func (r *MovieResolver) PosterPath() string {
	return r.r.PosterPath
}

// Year returns year
func (r *MovieResolver) Year() string {
	return r.r.YearAsString()
}

// Overview returns movie summary
func (r *MovieResolver) Overview() string {
	return r.r.Overview
}

// ImdbID returns imdb id
func (r *MovieResolver) ImdbID() string {
	return r.r.ImdbID
}

// TmdbID returns tmdb id
func (r *MovieResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}

// PlayState returns playstate for given user.
func (r *MovieResolver) PlayState() *PlayStateResolver {
	return &PlayStateResolver{r: r.r.PlayState}
}

// MovieFileResolver resolves the movie information
type MovieFileResolver struct {
	r db.MovieFile
}

// LibraryID returns library id
func (r *MovieFileResolver) LibraryID() int32 {
	// TODO: Will this be a problem if we ever run out of the 32int space?
	return int32(r.r.LibraryID)
}

// FilePath returns filesystem path
func (r *MovieFileResolver) FilePath() string {
	return r.r.FilePath
}

// FileName returns movie filename
func (r *MovieFileResolver) FileName() string {
	return r.r.FileName
}

// UUID returns movie uuid.
func (r *MovieFileResolver) UUID() string {
	return r.r.UUID
}

// TotalDuration returns the total duration in seconds based on the first encountered videostream.
func (r *MovieFileResolver) TotalDuration() *float64 {
	for _, stream := range r.r.Streams {
		if stream.StreamType == "video" {
			seconds := stream.TotalDuration.Seconds()
			return &seconds
		}
	}
	return nil
}

// Streams return all streams
func (r *MovieFileResolver) Streams() (streams []*StreamResolver) {
	for _, stream := range r.r.Streams {
		streams = append(streams, &StreamResolver{stream})
	}
	return streams
}
