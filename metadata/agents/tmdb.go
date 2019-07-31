package agents

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/ryanbradynd05/go-tmdb"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"strconv"
	"time"
)

const tmdbAPIKey = "0cdacd9ab172ac6ff69c8d84b2c938a8"

// TmdbAgent is a wrapper around themoviedb
type TmdbAgent struct {
	Tmdb *tmdb.TMDb
}

// NewTmdbAgent creates a new themoviedb agent.
func NewTmdbAgent() *TmdbAgent {
	return &TmdbAgent{tmdb.Init(tmdb.Config{
		APIKey:   tmdbAPIKey,
		Proxies:  nil,
		UseProxy: false,
	})}
}

func ParseTmdbDate(tmdbDate string) (time.Time, error) {
	return time.Parse("2006-01-02", tmdbDate)
}

// UpdateEpisodeMD updates the metadata information for the given episode.
func (a *TmdbAgent) UpdateEpisodeMD(episode *db.Episode, season *db.Season, series *db.Series) error {
	fullEpisode, err := a.Tmdb.GetTvEpisodeInfo(series.TmdbID, season.SeasonNumber, episode.EpisodeNum, nil)
	if err != nil {
		return err
	}

	if fullEpisode != nil {
		episode.AirDate = fullEpisode.AirDate
		episode.Name = fullEpisode.Name
		episode.TmdbID = fullEpisode.ID
		episode.Overview = fullEpisode.Overview
		episode.StillPath = fullEpisode.StillPath
		log.WithFields(log.Fields{"episodeName": episode.Name, "tmdbId": episode.TmdbID}).Debugln("Found episode metadata.")
		return nil
	}

	return fmt.Errorf("could not retrieve episode data from tmdb")
}

// UpdateSeasonMD updates the metadata information for the given season
func (a *TmdbAgent) UpdateSeasonMD(season *db.Season, series *db.Series) error {
	log.WithFields(log.Fields{"seasonNumber": season.SeasonNumber, "seriesName": series.Name}).Debugln("Looking for season metadata.")

	fullSeason, err := a.Tmdb.GetTvSeasonInfo(series.TmdbID, season.SeasonNumber, nil)
	if err == nil {
		season.AirDate = fullSeason.AirDate
		season.Overview = fullSeason.Overview
		season.Name = fullSeason.Name
		season.TmdbID = fullSeason.ID
		season.PosterPath = fullSeason.PosterPath
		log.WithFields(log.Fields{"seasonName": season.Name, "tmdbId": season.TmdbID}).Debugln("Found season metadata.")
	} else {
		log.WithFields(log.Fields{"error": err}).Warnln("Could not grab season information.")
		return err
	}
	return nil
}

// UpdateSeriesMD updates the metadata information for the given series.
func (a *TmdbAgent) UpdateSeriesMD(series *db.Series) error {
	if series.TmdbID == 0 {
		log.WithFields(log.Fields{"seriesName": series.Name}).Debugln("No TmdbID yet, looking for series metadata based on the parsed name.")
		var options = make(map[string]string)

		if series.FirstAirYear != 0 {
			options["first_air_date_year"] = strconv.FormatUint(series.FirstAirYear, 10)
		}
		searchRes, err := a.Tmdb.SearchTv(series.Name, options)

		if err != nil {
			return err
		}

		if len(searchRes.Results) > 0 {
			log.Debugln("Found Series that matches, using first result and doing deepscan.")
			tv := searchRes.Results[0] // Take the first result for now
			series.TmdbID = tv.ID
			series.FirstAirDate = tv.FirstAirDate
			series.OriginalName = tv.OriginalName
		}
	}

	fullTv, err := a.Tmdb.GetTvInfo(series.TmdbID, nil)
	if err == nil {
		log.WithFields(log.Fields{"seriesName": series.Name, "tmdbID": series.TmdbID}).Debugln("Updating metadata from tmdb agent.")
		series.Overview = fullTv.Overview
		series.Status = fullTv.Status
		series.Type = fullTv.Type
		series.BackdropPath = fullTv.BackdropPath
		series.PosterPath = fullTv.PosterPath
	} else {
		log.WithFields(log.Fields{"seriesName": series.Name, "tmdbID": series.TmdbID, "error": err}).Debugln("Could not grab full TV details.")
		return err
	}
	return nil
}

func (a *TmdbAgent) UpdateMovieMetadata(movie *db.Movie) error {
	r, err := a.Tmdb.GetMovieInfo(movie.TmdbID, nil)

	if err != nil {
		return errors.Wrap(err, "Failed to query TMDB for movie metadata")
	}

	movie.TmdbID = r.ID
	movie.Title = r.Title
	movie.OriginalTitle = r.OriginalTitle
	movie.ReleaseDate = r.ReleaseDate
	movie.Overview = r.Overview
	movie.BackdropPath = r.BackdropPath
	movie.PosterPath = r.PosterPath
	movie.ImdbID = r.ImdbID

	return nil
}

// TmdbSearchMovie directly exposes the TMDb search interface
func (a *TmdbAgent) TmdbSearchMovie(
	name string,
	options map[string]string,
) (*tmdb.MovieSearchResults, error) {
	return a.Tmdb.SearchMovie(name, options)
}

// TmdbSearchTv directly exposes the TMDb search interface
func (a *TmdbAgent) TmdbSearchTv(
	name string,
	options map[string]string,
) (*tmdb.TvSearchResults, error) {
	return a.Tmdb.SearchTv(name, options)
}
