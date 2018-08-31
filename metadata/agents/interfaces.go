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
