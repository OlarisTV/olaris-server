package agents

//go:generate counterfeiter . MetadataRetrievalAgent

import (
	"github.com/ryanbradynd05/go-tmdb"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// MetadataRetrievalAgent can retrieve metadata for media items.
type MetadataRetrievalAgent interface {
	UpdateMovieMD(movie *db.Movie, tmdbID int) error
	UpdateSeasonMD(season *db.Season, seriesTmdbID int, seasonNum int) error
	UpdateEpisodeMD(episode *db.Episode, seriesTmdbID int, seasonNum int, episodeNum int) error
	UpdateSeriesMD(series *db.Series, tmdbID int) error
	// TODO(Leon Handreke): This totally breaks the abstraction, but we need the interface
	//  to be able to fake it.
	TmdbSearchMovie(name string, options map[string]string) (*tmdb.MovieSearchResults, error)
	TmdbSearchTv(name string, options map[string]string) (*tmdb.TvSearchResults, error)
}
