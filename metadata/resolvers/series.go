package resolvers

import (
	"context"
	"strconv"

	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

type mustUUIDArgs struct {
	UUID *string
}

// Episode returns episode.
func (r *Resolver) Episode(ctx context.Context, args *mustUUIDArgs) *EpisodeResolver {
	episode, err := db.FindEpisodeByUUID(*args.UUID)
	// TODO(Maran): return an actual error to the client, not just an empty dict
	if err == nil {
		return &EpisodeResolver{r: *episode}
	}
	return &EpisodeResolver{r: db.Episode{}}

}

// Season returns season.
func (r *Resolver) Season(ctx context.Context, args *mustUUIDArgs) *SeasonResolver {
	season, _ := db.FindSeasonByUUID(*args.UUID)
	return &SeasonResolver{r: *season}
}

// Series return series.
func (r *Resolver) Series(ctx context.Context, args *queryArgs) []*SeriesResolver {
	var series []*db.Series

	if args.UUID != nil {
		serie, err := db.FindSeriesByUUID(*args.UUID)
		if err != nil {
			series = []*db.Series{}
		} else {
			series = []*db.Series{serie}
		}
	} else {
		qd := createQd(args)
		series, _ = db.FindAllSeries(qd)
	}

	var resolvers []*SeriesResolver
	for _, s := range series {
		resolvers = append(resolvers, &SeriesResolver{r: *s})
	}

	return resolvers
}

// SeriesResolver resolvers a serie.
type SeriesResolver struct {
	r db.Series
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
func (r *SeriesResolver) UnwatchedEpisodesCount(ctx context.Context) int32 {
	userID, _ := auth.UserID(ctx)
	epCount := db.UnwatchedEpisodesInSeriesCount(r.r.ID, userID)
	return int32(epCount)
}

// Seasons returns all seasons.
func (r *SeriesResolver) Seasons() []*SeasonResolver {
	var seasons []*SeasonResolver

	for _, season := range db.FindSeasonsForSeries(r.r.ID) {
		seasons = append(seasons, &SeasonResolver{r: season})
	}

	return seasons
}

// SeasonResolver resolves season
type SeasonResolver struct {
	r db.Season
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
func (r *SeasonResolver) UnwatchedEpisodesCount(ctx context.Context) int32 {
	userID, _ := auth.UserID(ctx)
	return int32(db.UnwatchedEpisodesInSeasonCount(r.r.ID, userID))
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
	series, _ := db.FindSeries(r.r.SeriesID)
	return &SeriesResolver{*series}
}

// Episodes returns seasonal episodes.
func (r *SeasonResolver) Episodes() []*EpisodeResolver {
	var res []*EpisodeResolver
	for _, episode := range db.FindEpisodesForSeason(r.r.ID) {
		res = append(res, &EpisodeResolver{r: episode})
	}
	return res
}

// EpisodeResolver resolves episode.
type EpisodeResolver struct {
	r db.Episode
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
	s, _ := db.FindSeason(r.r.SeasonID)
	return &SeasonResolver{*s}
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
func (r *EpisodeResolver) PlayState(ctx context.Context) *PlayStateResolver {
	userID, _ := auth.UserID(ctx)
	playState, _ := db.FindPlayState(r.r.UUID, userID)
	if playState == nil {
		playState = &db.PlayState{}
	}
	return &PlayStateResolver{r: *playState}
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
func (r *EpisodeFileResolver) FilePath() (string, error) {
	fileLocator, err := filesystem.ParseFileLocator(r.r.FilePath)
	if err != nil {
		return "", err
	}
	return fileLocator.Path, nil
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
func (r *EpisodeFileResolver) FileSize() string {
	return strconv.FormatInt(r.r.Size, 10)
}

// Library returns library
func (r *EpisodeFileResolver) Library() *LibraryResolver {
	lib := db.FindLibrary(int(r.r.LibraryID))
	return &LibraryResolver{r: Library{Library: lib}}
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
