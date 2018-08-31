package agents

import (
	"fmt"
	"github.com/ryanbradynd05/go-tmdb"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"strconv"
)

const tmdbAPIKey = "0cdacd9ab172ac6ff69c8d84b2c938a8"

// TmdbAgent is a wrapper around themoviedb
type TmdbAgent struct {
	Tmdb *tmdb.TMDb
}

// NewTmdbAgent creates a new themoviedb agent.
func NewTmdbAgent() *TmdbAgent {
	return &TmdbAgent{tmdb.Init(tmdbAPIKey)}

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
		log.WithFields(log.Fields{"error": err}).Warnln("Could not grab seasonal information.")
		return err
	}
	return nil
}

// UpdateSeriesMD updates the metadata information for the given series.
func (a *TmdbAgent) UpdateSeriesMD(series *db.Series) error {
	log.WithFields(log.Fields{"seriesName": series.Name}).Debugln("Looking for series metadata.")
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
		fullTv, err := a.Tmdb.GetTvInfo(tv.ID, nil)
		if err == nil {
			series.Overview = fullTv.Overview
			series.Status = fullTv.Status
			series.Type = fullTv.Type
		} else {
			log.Warnln("Could not get full results, only adding search results. Error:", err)
		}
		series.TmdbID = tv.ID
		series.FirstAirDate = tv.FirstAirDate
		series.OriginalName = tv.OriginalName
		series.BackdropPath = tv.BackdropPath
		series.PosterPath = tv.PosterPath
		log.WithFields(log.Fields{"seriesName": series.Name, "tmdbID": tv.ID}).Debugln("Found series metadata.")
	}
	return nil
}

// UpdateMovieMD the given movie with Metadata from themoviedatabase.org
func (a *TmdbAgent) UpdateMovieMD(movie *db.Movie) error {
	var options = make(map[string]string)
	if movie.Year > 0 {
		options["year"] = movie.YearAsString()
	}
	searchRes, err := a.Tmdb.SearchMovie(movie.Title, options)

	if err != nil {
		return err
	}

	if len(searchRes.Results) > 0 {
		log.Debugln("Found movie that matches, using first result from search and requesting more movie details.")
		mov := searchRes.Results[0] // Take the first result for now
		fullMov, err := a.Tmdb.GetMovieInfo(mov.ID, nil)
		if err == nil {
			movie.Overview = fullMov.Overview
			movie.ImdbID = fullMov.ImdbID
		} else {
			log.Warnln("Could not get full results, only adding search results. Error:", err)
		}
		movie.TmdbID = mov.ID
		movie.ReleaseDate = mov.ReleaseDate
		movie.OriginalTitle = mov.OriginalTitle
		movie.BackdropPath = mov.BackdropPath
		movie.PosterPath = mov.PosterPath
		//	env.Db.Save(&movie)
		log.WithFields(movie.LogFields()).Println("identified movie.")
	} else {
		log.WithFields(log.Fields{
			"title": movie.Title,
		}).Warnln("Could not find match based on parsed title.")
	}
	return nil
}
