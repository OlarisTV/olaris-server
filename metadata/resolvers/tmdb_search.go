package resolvers

import (
	"context"
	"github.com/ryanbradynd05/go-tmdb"
	"gitlab.com/olaris/olaris-server/metadata/agents"
)

type tmdbSearchMoviesArgs struct {
	Query string
}

func (r *Resolver) TmdbSearchMovies(ctx context.Context,
	args *tmdbSearchMoviesArgs) ([]*TmdbMovieSearchItemResolver, error) {

	searchRes, err := r.env.MetadataRetrievalAgent.TmdbSearchMovie(args.Query, nil)
	if err != nil {
		return nil, err
	}

	var res []*TmdbMovieSearchItemResolver
	for _, movieResult := range searchRes.Results {
		res = append(res, &TmdbMovieSearchItemResolver{r: movieResult})
	}
	return res, nil
}

type TmdbMovieSearchItemResolver struct {
	r tmdb.MovieShort
}

func (r *TmdbMovieSearchItemResolver) Title() string {
	return r.r.Title
}

func (r *TmdbMovieSearchItemResolver) ReleaseYear() (int32, error) {
	releaseDate, err := agents.ParseTmdbDate(r.r.ReleaseDate)
	if err != nil {
		return 0, err
	}
	return int32(releaseDate.Year()), nil
}

func (r *TmdbMovieSearchItemResolver) Overview() string {
	return r.r.Overview
}

func (r *TmdbMovieSearchItemResolver) TmdbID() int32 {
	return int32(r.r.ID)
}

func (r *TmdbMovieSearchItemResolver) BackdropPath() string {
	return r.r.BackdropPath
}

func (r *TmdbMovieSearchItemResolver) PosterPath() string {
	return r.r.PosterPath
}
