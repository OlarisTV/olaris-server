package agents

import (
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// MovieAgent is an interface that determines it can retrieve movie metadata.
type MovieAgent interface {
	UpdateMovieMD(*db.Movie) error
}

// SeasonAgent is an interface that determines it can retrieve season metadata.
type SeasonAgent interface {
	UpdateSeasonMD(*db.Season, *db.Series) error
}

// EpisodeAgent is an interface that determines it can retrieve episode metadata.
type EpisodeAgent interface {
	UpdateEpisodeMD(*db.Episode, *db.Season, *db.Series) error
}

// SeriesAgent is an interface that determines it can retrieve episode metadata.
type SeriesAgent interface {
	UpdateSeriesMD(*db.Series) error
}

// UpdateMovieMD is a generic method that invokes an agent and updates the metadata.
func UpdateMovieMD(mi MovieAgent, movie *db.Movie) error {
	return mi.UpdateMovieMD(movie)
}

// UpdateEpisodeMD is a generic method that invokes an agent and updates the metadata
func UpdateEpisodeMD(a EpisodeAgent, episode *db.Episode, season *db.Season, series *db.Series) error {
	return a.UpdateEpisodeMD(episode, season, series)
}

// UpdateSeasonMD is a generic method that invokes an agent and updates the metadata
func UpdateSeasonMD(a SeasonAgent, season *db.Season, series *db.Series) error {
	return a.UpdateSeasonMD(season, series)
}

// UpdateSeriesMD is a generic method that invokes an agent and updates the metadata
func UpdateSeriesMD(a SeriesAgent, series *db.Series) error {
	return a.UpdateSeriesMD(series)
}
