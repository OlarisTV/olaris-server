package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

type mustUUIDArgs struct {
	UUID *string
}

// Season wrapper object around db.Season
type Season struct {
	db.Season
	UnwatchedEpisodeCount uint
	UserID                uint
}

// Series wrapper object around db.Series so it can hold the userid
type Series struct {
	db.Series
	UnwatchedEpisodeCount uint
	// TODO: Figure out if this is racey
	UserID uint
}

// Episode is a wrapper object around db.Episode
type Episode struct {
	db.Episode
	UserID uint
}

func newEpisode(dbEpisode *db.Episode, userID uint) Episode {
	return Episode{*dbEpisode, userID}
}
func newSeason(dbSeason *db.Season, userID uint) Season {
	return Season{*dbSeason, 0, userID}
}
func newSeries(dbSeries *db.Series, userID uint) Series {
	return Series{*dbSeries, 0, userID}
}

// Episode returns episode.
func (r *Resolver) Episode(ctx context.Context, args *mustUUIDArgs) *EpisodeResolver {
	userID, _ := auth.UserID(ctx)
	dbepisode := db.FindEpisodeByUUID(*args.UUID, userID)
	if dbepisode.ID != 0 {
		ep := newEpisode(&dbepisode, userID)
		return &EpisodeResolver{r: ep}
	}
	return &EpisodeResolver{r: Episode{}}

}

// Season returns season.
func (r *Resolver) Season(ctx context.Context, args *mustUUIDArgs) *SeasonResolver {
	userID, _ := auth.UserID(ctx)
	dbseason := db.FindSeasonByUUID(*args.UUID)
	season := newSeason(&dbseason, userID)

	return &SeasonResolver{r: season}
}

// Series return series.
func (r *Resolver) Series(ctx context.Context, args *queryArgs) []*SeriesResolver {
	userID, _ := auth.UserID(ctx)
	var resolvers []*SeriesResolver
	var series []db.Series

	qd := createQd(args)

	if args.UUID != nil {
		series = db.FindSeriesByUUID(*args.UUID)
	} else {
		series = db.FindAllSeries(qd)
	}

	for _, serie := range series {
		s := newSeries(&serie, userID)

		res := SeriesResolver{r: s}

		resolvers = append(resolvers, &res)
	}

	return resolvers
}

// SeriesResolver resolvers a serie.
type SeriesResolver struct {
	r Series
}

// Name returns name.
func (r *SeriesResolver) Name() string {
	return r.r.Name
}

// UUID returns uuid.
func (r *SeriesResolver) UUID() string {
	return r.r.UUID
}

// Overview returns overview.
func (r *SeriesResolver) Overview() string {
	return r.r.Overview
}

// FirstAirDate returns air date.
func (r *SeriesResolver) FirstAirDate() string {
	return r.r.FirstAirDate
}

// Status returns serie status.
func (r *SeriesResolver) Status() string {
	return r.r.Status
}

// Type returns content type.
func (r *SeriesResolver) Type() string {
	return r.r.Type
}

// PosterPath resturn uri to poster.
func (r *SeriesResolver) PosterPath() string {
	return r.r.PosterPath
}

// BackdropPath returns uri to backdrop.
func (r *SeriesResolver) BackdropPath() string {
	return r.r.BackdropPath
}

// TmdbID returns tmdb id
func (r *SeriesResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}

// UnwatchedEpisodesCount returns the amount of unwatched episodes for the given season
func (r *SeriesResolver) UnwatchedEpisodesCount() int32 {
	epCount := db.UnwatchedEpisodesInSeriesCount(r.r.ID, r.r.UserID)
	return int32(epCount)
}

// Seasons returns all seasons.
func (r *SeriesResolver) Seasons() []*SeasonResolver {
	var seasons []*SeasonResolver

	for _, dbseason := range db.FindSeasonsForSeries(r.r.ID) {
		season := newSeason(&dbseason, r.r.UserID)
		seasons = append(seasons, &SeasonResolver{r: season})
	}

	return seasons
}

// SeasonResolver resolves season
type SeasonResolver struct {
	r Season
}

// Name returns name.
func (r *SeasonResolver) Name() string {
	return r.r.Name
}

// UUID returns uuid.
func (r *SeasonResolver) UUID() string {
	return r.r.UUID
}

// Overview returns season overview.
func (r *SeasonResolver) Overview() string {
	return r.r.Overview
}

// AirDate returns seasonal air date.
func (r *SeasonResolver) AirDate() string {
	return r.r.AirDate
}

// PosterPath resturn uri to poster.
func (r *SeasonResolver) PosterPath() string {
	return r.r.PosterPath
}

// UnwatchedEpisodesCount returns the amount of unwatched episodes for the given season
func (r *SeasonResolver) UnwatchedEpisodesCount() int32 {
	return int32(db.UnwatchedEpisodesInSeasonCount(r.r.ID, r.r.UserID))
}

// TmdbID returns tmdb id.
func (r *SeasonResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}

// SeasonNumber returns season number.
func (r *SeasonResolver) SeasonNumber() int32 {
	return int32(r.r.SeasonNumber)
}

// Series returns the series this season belongs to.
func (r *SeasonResolver) Series() *SeriesResolver {
	s := db.FindSeries(r.r.SeriesID)
	series := newSeries(&s, r.r.UserID)
	return &SeriesResolver{series}
}

// Episodes returns seasonal episodes.
func (r *SeasonResolver) Episodes() []*EpisodeResolver {
	var eps []*EpisodeResolver
	for _, episode := range db.FindEpisodesForSeason(r.r.ID, r.r.UserID) {
		epp := newEpisode(&episode, r.r.UserID)
		ep := &EpisodeResolver{r: epp}
		eps = append(eps, ep)
	}
	return eps
}

// EpisodeResolver resolves episode.
type EpisodeResolver struct {
	r Episode
}

// Files return all files for this episode.
func (r *EpisodeResolver) Files() (files []*EpisodeFileResolver) {
	for _, episode := range r.r.EpisodeFiles {
		files = append(files, &EpisodeFileResolver{r: episode})
	}
	return files
}

// Name returns name.
func (r *EpisodeResolver) Name() string {
	return r.r.Name
}

// Season returns the season the episode belongs to.
func (r *EpisodeResolver) Season() *SeasonResolver {
	s := db.FindSeason(r.r.SeasonID)
	season := newSeason(&s, r.r.UserID)
	return &SeasonResolver{season}
}

// UUID returns uuid.
func (r *EpisodeResolver) UUID() string {
	return r.r.UUID
}

// Overview returns overview.
func (r *EpisodeResolver) Overview() string {
	return r.r.Overview
}

// AirDate returns air date.
func (r *EpisodeResolver) AirDate() string {
	return r.r.AirDate
}

// StillPath returns uri to still image.
func (r *EpisodeResolver) StillPath() string {
	return r.r.StillPath
}

// TmdbID returns tmdb id.
func (r *EpisodeResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}

// EpisodeNumber returns episode number.
func (r *EpisodeResolver) EpisodeNumber() int32 {
	return int32(r.r.EpisodeNum)
}

// PlayState returns episode playstate information.
func (r *EpisodeResolver) PlayState() *PlayStateResolver {
	ps := db.FindPlaystateForEpisode(r.r.ID, r.r.UserID)
	return &PlayStateResolver{r: ps}
}

// PlayStateResolver resolves playstate
type PlayStateResolver struct {
	r db.PlayState
}

// UUID returns UUID of the mediaItem
func (res *PlayStateResolver) UUID() string {
	return res.r.UUID
}

// Finished returns a bool when content has been watched.
func (res *PlayStateResolver) Finished() bool {
	return res.r.Finished
}

// Playtime current playtime.
func (res *PlayStateResolver) Playtime() float64 {
	return res.r.Playtime
}

// EpisodeFileResolver resolves episodefile.
type EpisodeFileResolver struct {
	r db.EpisodeFile
}

// FilePath returns filesystem path to file.
func (r *EpisodeFileResolver) FilePath() string {
	return r.r.FilePath
}

// FileName returns filename.
func (r *EpisodeFileResolver) FileName() string {
	return r.r.FileName
}

// UUID returns uuid.
func (r *EpisodeFileResolver) UUID() string {
	return r.r.UUID
}

// FileSize returns episode filesize
func (r *EpisodeFileResolver) FileSize() int32 {
	return int32(r.r.Size)
}

// TotalDuration returns the total duration in seconds based on the first encountered videostream.
func (r *EpisodeFileResolver) TotalDuration() *float64 {
	for _, stream := range r.r.Streams {
		if stream.StreamType == "video" {
			seconds := stream.TotalDuration.Seconds()
			return &seconds
		}
	}
	return nil
}

// Streams return stream information.
func (r *EpisodeFileResolver) Streams() (streams []*StreamResolver) {
	for _, stream := range r.r.Streams {
		streams = append(streams, &StreamResolver{r: stream})
	}
	return streams
}
