package resolvers

import (
	"context"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
)

type Season struct {
	db.Season
	Episodes []*EpisodeResolver
}

type Series struct {
	db.Series
	Seasons []*SeasonResolver
}

func (r *Resolver) Episode(ctx context.Context, args *MustUuidArgs) *EpisodeResolver {
	userID := helpers.GetUserID(ctx)
	dbepisode := db.FindEpisodeByUUID(&args.Uuid, userID)
	if dbepisode != nil {
		return &EpisodeResolver{r: *dbepisode}
	} else {
		return &EpisodeResolver{r: db.Episode{}}
	}

}

func (r *Resolver) Season(ctx context.Context, args *MustUuidArgs) *SeasonResolver {
	userID := helpers.GetUserID(ctx)
	dbseason := db.FindSeasonByUUID(&args.Uuid)
	season := Season{dbseason, nil}

	// TODO(Maran): This part can be DRIED up and moved into it's own function
	for _, episode := range db.FindEpisodesForSeason(season.ID, userID) {
		season.Episodes = append(season.Episodes, &EpisodeResolver{r: episode})
	}

	return &SeasonResolver{r: season}
}

func (r *Resolver) Series(ctx context.Context, args *UuidArgs) []*SeriesResolver {
	userID := helpers.GetUserID(ctx)
	var resolvers []*SeriesResolver
	var series []db.Series

	if args.Uuid != nil {
		series = db.FindSeriesByUUID(args.Uuid)
	} else {
		series = db.FindAllSeries()
	}

	for _, serie := range series {
		serieResolver := CreateSeriesResolver(serie, userID)
		resolvers = append(resolvers, serieResolver)
	}

	return resolvers
}

func CreateSeriesResolver(dbserie db.Series, userID uint) *SeriesResolver {
	serie := Series{dbserie, nil}
	for _, dbseason := range db.FindSeasonsForSeries(serie.ID) {
		season := Season{dbseason, nil}
		for _, episode := range db.FindEpisodesForSeason(season.ID, userID) {
			season.Episodes = append(season.Episodes, &EpisodeResolver{r: episode})
		}
		serie.Seasons = append(serie.Seasons, &SeasonResolver{r: season})
	}
	return &SeriesResolver{r: serie}
}

type SeriesResolver struct {
	r Series
}

func (r *SeriesResolver) Name() string {
	return r.r.Name
}
func (r *SeriesResolver) UUID() string {
	return r.r.UUID
}
func (r *SeriesResolver) Overview() string {
	return r.r.Overview
}
func (r *SeriesResolver) FirstAirDate() string {
	return r.r.FirstAirDate
}
func (r *SeriesResolver) Status() string {
	return r.r.Status
}
func (r *SeriesResolver) Type() string {
	return r.r.Type
}
func (r *SeriesResolver) PosterPath() string {
	return r.r.PosterPath
}
func (r *SeriesResolver) BackdropPath() string {
	return r.r.BackdropPath
}
func (r *SeriesResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}
func (r *SeriesResolver) Seasons() []*SeasonResolver {
	return r.r.Seasons
}

type SeasonResolver struct {
	r Season
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
	r db.Episode
}

func (r *EpisodeResolver) Files() (files []*EpisodeFileResolver) {
	for _, episode := range r.r.EpisodeFiles {
		files = append(files, &EpisodeFileResolver{r: episode})
	}
	return files
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
func (r *EpisodeResolver) EpisodeNumber() int32 {
	return int32(r.r.EpisodeNum)
}
func (r *EpisodeResolver) PlayState() *PlayStateResolver {
	return &PlayStateResolver{r: r.r.PlayState}
}

type PlayStateResolver struct {
	r db.PlayState
}

func (r *PlayStateResolver) Finished() bool {
	return r.r.Finished
}
func (r *PlayStateResolver) Playtime() float64 {
	return r.r.Playtime
}

type EpisodeFileResolver struct {
	r db.EpisodeFile
}

func (r *EpisodeFileResolver) FilePath() string {
	return r.r.FilePath
}
func (r *EpisodeFileResolver) FileName() string {
	return r.r.FileName
}
func (r *EpisodeFileResolver) UUID() string {
	return r.r.UUID
}
func (r *EpisodeFileResolver) Streams() (streams []*StreamResolver) {
	for _, stream := range r.r.Streams {
		streams = append(streams, &StreamResolver{stream})
	}
	return streams
}
