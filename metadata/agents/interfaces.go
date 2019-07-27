package agents

//go:generate counterfeiter . MetadataRetrievalAgent

import (
	"github.com/ryanbradynd05/go-tmdb"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// MetadataRetrievalAgent can retrieve metadata for media items.
type MetadataRetrievalAgent interface {
	UpdateMovieMD(*db.Movie) error
	UpdateMovieMetadataFromTmdbID(movie *db.Movie, tmdbID int) error
	UpdateSeasonMD(*db.Season, *db.Series) error
	UpdateEpisodeMD(*db.Episode, *db.Season, *db.Series) error
	UpdateSeriesMD(*db.Series) error
	// TODO(Leon Handreke): This totally breaks the abstraction, but we need the interface
	//  to be able to fake it.
	TmdbSearchMovie(name string, options map[string]string) (*tmdb.MovieSearchResults, error)
}
