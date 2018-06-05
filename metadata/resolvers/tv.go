package resolvers

import (
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
)

type TvSeason struct {
	db.TvSeason
	Episodes []*EpisodeResolver
}

type TvSeries struct {
	db.TvSeries
	Seasons []*SeasonResolver
}

func (r *Resolver) TvSeries() []*TvSeriesResolver {
	var resolvers []*TvSeriesResolver
	var series []db.TvSeries
	r.ctx.Db.Find(&series)
	for _, serie := range series {
		serieResolver := CreateSeriesResolver(r.ctx, serie)
		resolvers = append(resolvers, serieResolver)

	}
	return resolvers
}

func CreateSeriesResolver(ctx *db.MetadataContext, dbserie db.TvSeries) *TvSeriesResolver {
	serie := TvSeries{dbserie, nil}
	var seasons []db.TvSeason
	ctx.Db.Where("tv_series_id = ?", serie.ID).Find(&seasons)
	for _, dbseason := range seasons {
		season := TvSeason{dbseason, nil}
		var episodes []db.TvEpisode
		ctx.Db.Where("tv_season_id = ?", season.ID).Find(&episodes)
		for _, episode := range episodes {
			season.Episodes = append(season.Episodes, &EpisodeResolver{r: episode})
		}
		serie.Seasons = append(serie.Seasons, &SeasonResolver{r: season})
	}
	return &TvSeriesResolver{r: serie}
}

type TvSeriesResolver struct {
	r TvSeries
}

func (r *TvSeriesResolver) Name() string {
	return r.r.Name
}
func (r *TvSeriesResolver) UUID() string {
	return r.r.UUID
}
func (r *TvSeriesResolver) Overview() string {
	return r.r.Overview
}
func (r *TvSeriesResolver) FirstAirDate() string {
	return r.r.FirstAirDate
}
func (r *TvSeriesResolver) Status() string {
	return r.r.Status
}
func (r *TvSeriesResolver) Type() string {
	return r.r.Type
}
func (r *TvSeriesResolver) PosterPath() string {
	return r.r.PosterPath
}
func (r *TvSeriesResolver) BackdropPath() string {
	return r.r.BackdropPath
}
func (r *TvSeriesResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}
func (r *TvSeriesResolver) Seasons() []*SeasonResolver {
	return r.r.Seasons
}

type SeasonResolver struct {
	r TvSeason
}

func (r *SeasonResolver) Name() string {
	return r.r.Name
}

func (r *SeasonResolver) UUID() string {
	return r.r.UUID
}
func (r *SeasonResolver) Overview() string {
	return r.r.Overview
}
func (r *SeasonResolver) AirDate() string {
	return r.r.AirDate
}
func (r *SeasonResolver) PosterPath() string {
	return r.r.PosterPath
}
func (r *SeasonResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}

func (r *SeasonResolver) SeasonNumber() int32 {
	return int32(r.r.SeasonNumber)
}
func (r *SeasonResolver) Episodes() []*EpisodeResolver {
	return r.r.Episodes
}

type EpisodeResolver struct {
	r db.TvEpisode
}

func (r *EpisodeResolver) Name() string {
	return r.r.Name
}

func (r *EpisodeResolver) UUID() string {
	return r.r.UUID
}

func (r *EpisodeResolver) Overview() string {
	return r.r.Overview
}
func (r *EpisodeResolver) AirDate() string {
	return r.r.AirDate
}
func (r *EpisodeResolver) StillPath() string {
	return r.r.StillPath
}
func (r *EpisodeResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}
func (r *EpisodeResolver) FilePath() string {
	return r.r.FilePath
}
func (r *EpisodeResolver) FileName() string {
	return r.r.FileName
}
func (r *EpisodeResolver) EpisodeNumber() string {
	return r.r.EpisodeNum
}
