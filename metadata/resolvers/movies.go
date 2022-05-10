package resolvers

import (
	"context"
	"fmt"
	"strconv"

	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

type movieQueryArgs struct {
	queryArgs
	Sort *MovieSort
}

func (m *movieQueryArgs) asQueryDetails() *db.QueryDetails {
	qd := m.queryArgs.asQueryDetails()

	if m.Sort != nil {
		qd.SortColumn = m.Sort.ToString()
	}

	return qd
}

type posterURLArgs struct {
	Width int32
}

// Movies returns all movies.
func (r *Resolver) Movies(ctx context.Context, args *movieQueryArgs) []*MovieResolver {
	var l []*MovieResolver
	var movies []db.Movie
	qd := args.asQueryDetails()
	if args.UUID != nil {
		movie, _ := db.FindMovieByUUID(*args.UUID)
		movies = []db.Movie{*movie}
	} else {
		movies = db.FindAllMovies(qd)
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
	for _, file := range db.FindFilesForMovieUUID(r.r.UUID) {
		resolver := MovieFileResolver{r: *file}
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

// PosterURL returns poster's URL for the given size
func (r *MovieResolver) PosterURL(ctx context.Context, args *posterURLArgs) string {
	actualWidth := "original"
	if args.Width > 0 {
		// TODO: get this dynamically from TMDB at server startup
		// (or maybe it's already fetched somewhere?)
		availableWidths := []int32{92, 154, 185, 342, 500, 780}
		for _, currentWidth := range availableWidths {
			if currentWidth >= args.Width {
				actualWidth = fmt.Sprintf("w%d", currentWidth)
				break
			}
		}
	}
	return fmt.Sprintf("/olaris/m/images/tmdb/%s%s", actualWidth, r.r.PosterPath)
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
func (r *MovieResolver) PlayState(ctx context.Context) *PlayStateResolver {
	userID, _ := auth.UserID(ctx)
	playState, _ := db.FindPlayState(r.r.UUID, userID)
	if playState == nil {
		playState = &db.PlayState{}
	}
	return &PlayStateResolver{r: *playState}
}

// MovieFileResolver resolves the movie information
type MovieFileResolver struct {
	r db.MovieFile
}

// Library returns library
func (r *MovieFileResolver) Library() *LibraryResolver {
	lib := db.FindLibrary(int(r.r.LibraryID))
	return &LibraryResolver{r: Library{Library: lib}}
}

// Episode returns the episode this file belongs to
func (r *MovieFileResolver) Movie() (e *MovieResolver) {
	movie, err := db.FindMovieByID(r.r.MovieID)
	if err == nil {
		e = &MovieResolver{r: *movie}
	}
	return e
}

// LibraryID returns library id
func (r *MovieFileResolver) LibraryID() int32 {
	// TODO: Will this be a problem if we ever run out of the 32int space?
	return int32(r.r.LibraryID)
}

// FilePath returns filesystem path
func (r *MovieFileResolver) FilePath() (string, error) {
	fileLocator, err := filesystem.ParseFileLocator(r.r.FilePath)
	if err != nil {
		return "", err
	}
	return fileLocator.Path, nil
}

// FileName returns movie filename
func (r *MovieFileResolver) FileName() string {
	return r.r.FileName
}

// FileSize returns movie filesize
func (r *MovieFileResolver) FileSize() string {
	return strconv.FormatInt(r.r.Size, 10)
}

// UUID returns movie uuid.
func (r *MovieFileResolver) UUID() string {
	return r.r.UUID
}

// TotalDuration returns the total duration in seconds based on the first encountered videostream.
func (r *MovieFileResolver) TotalDuration() *float64 {
	for _, stream := range db.FindStreamsForMovieFileUUID(r.r.UUID) {
		if stream.StreamType == "video" {
			seconds := stream.TotalDuration.Seconds()
			return &seconds
		}
	}
	return nil
}

// Streams return all streams
func (r *MovieFileResolver) Streams() (streams []*StreamResolver) {
	for _, stream := range db.FindStreamsForMovieFileUUID(r.r.UUID) {
		streams = append(streams, &StreamResolver{r: *stream})
	}
	return streams
}
