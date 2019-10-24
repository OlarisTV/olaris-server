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

func (r *TmdbMovieSearchItemResolver) ReleaseYear() (*int32, error) {
	if r.r.ReleaseDate == "" {
		return nil, nil
	}

	releaseDate, err := agents.ParseTmdbDate(r.r.ReleaseDate)
	if err != nil {
		return nil, err
	}
	releaseYear := int32(releaseDate.Year())
	return &releaseYear, nil
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

type tmdbSearchSeriesArgs struct {
	Query string
}

func (r *Resolver) TmdbSearchSeries(ctx context.Context,
	args *tmdbSearchSeriesArgs) ([]*TmdbSeriesSearchItemResolver, error) {

	searchRes, err := r.env.MetadataRetrievalAgent.TmdbSearchTv(args.Query, nil)
	if err != nil {
		return nil, err
	}

	var res []*TmdbSeriesSearchItemResolver
	for _, seriesResult := range searchRes.Results {
		res = append(res, &TmdbSeriesSearchItemResolver{r: seriesResult})
	}
	return res, nil
}

type TmdbSeriesSearchItemResolver struct {
	// This struct is copied from tmdb.TvSearchResults, it doesn't have its own
	// type name.
	r struct {
		BackdropPath  string `json:"backdrop_path"`
		ID            int
		OriginalName  string   `json:"original_name"`
		FirstAirDate  string   `json:"first_air_date"`
		OriginCountry []string `json:"origin_country"`
		PosterPath    string   `json:"poster_path"`
		Popularity    float32
		Name          string
		VoteAverage   float32 `json:"vote_average"`
		VoteCount     uint32  `json:"vote_count"`
	}
}

func (r *TmdbSeriesSearchItemResolver) Name() string {
	return r.r.Name
}

func (r *TmdbSeriesSearchItemResolver) FirstAirYear() (*int32, error) {
	if r.r.FirstAirDate == "" {
		return nil, nil
	}
	firstAirDate, err := agents.ParseTmdbDate(r.r.FirstAirDate)
	if err != nil {
		return nil, err
	}
	firstAirYear := int32(firstAirDate.Year())
	return &firstAirYear, nil
}

func (r *TmdbSeriesSearchItemResolver) TmdbID() int32 {
	return int32(r.r.ID)
}

func (r *TmdbSeriesSearchItemResolver) BackdropPath() string {
	return r.r.BackdropPath
}

func (r *TmdbSeriesSearchItemResolver) PosterPath() string {
	return r.r.PosterPath
}
