package metadata

import (
	"errors"
	"github.com/ryanbradynd05/go-tmdb"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers/levenshtein"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/parsers"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

// ForceMovieMetadataUpdate refreshes all metadata for all movies
func (m *MetadataManager) ForceMovieMetadataUpdate() {
	for _, movie := range db.FindAllMovies(nil) {
		m.UpdateMovieMD(&movie)
	}
}

// UpdateMovieMD updates the database record with the latest data from the agent
func (m *MetadataManager) UpdateMovieMD(movie *db.Movie) error {
	log.WithFields(log.Fields{"title": movie.Title}).
		Println("Refreshing metadata for movie.")

	if err := m.agent.UpdateMovieMD(movie, movie.TmdbID); err != nil {
		return err
	}
	// TODO(Leon Handreke): return an error here.
	db.SaveMovie(movie)
	return nil
}

// GetOrCreateMovieForMovieFile tries to create a Movie object by parsing the filename of the
// given MovieFile and looking it up in TMDB. It associates the MovieFile with the new Model.
// If no matching movie can be found in TMDB, it returns an error.
func (m *MetadataManager) GetOrCreateMovieForMovieFile(
	movieFile *db.MovieFile) (*db.Movie, error) {

	// If we already have an associated movie, don't create a new one
	if movieFile.MovieID != 0 {
		return db.FindMovieByID(movieFile.MovieID)
	}

	name := strings.TrimSuffix(movieFile.FileName, filepath.Ext(movieFile.FileName))
	parsedInfo := parsers.ParseMovieName(name)

	var options = make(map[string]string)
	if parsedInfo.Year > 0 {
		options["year"] = strconv.FormatUint(parsedInfo.Year, 10)
	}
	searchRes, err := m.agent.TmdbSearchMovie(parsedInfo.Title, options)
	if err != nil {
		return nil, err
	}

	if len(searchRes.Results) == 0 {
		log.WithFields(log.Fields{
			"title": parsedInfo.Title,
			"year":  parsedInfo.Year,
		}).Warnln("Could not find match based on parsed title and given year.")

		return nil, errors.New("Could not find match in TMDB ID for given filename")
	}

	log.Debugln("Found movie that matches, using first result from search and requesting more movie details.")

	var bestDistance = math.MaxInt32
	var bestResult tmdb.MovieShort
	for _, r := range searchRes.Results {
		d := levenshtein.ComputeDistance(parsedInfo.Title, r.Title)
		if d < bestDistance {
			bestDistance = d
			bestResult = r
		}
	}

	movie, err := m.GetOrCreateMovieByTmdbID(bestResult.ID)
	if err != nil {
		return nil, err
	}

	movieFile.Movie = *movie
	db.SaveMovieFile(movieFile)

	movie.MovieFiles = []db.MovieFile{*movieFile}
	return movie, nil
}

// GetOrCreateMovieByTmdbID gets or creates a Movie object in the database,
// populating it with the details of the movie indicated by the TMDB ID.
func (m *MetadataManager) GetOrCreateMovieByTmdbID(tmdbID int) (*db.Movie, error) {

	// Lock so that we don't create the same movie twice
	m.moviesCreationMutex.Lock()
	defer m.moviesCreationMutex.Unlock()

	movie, err := db.FindMovieByTmdbID(tmdbID)
	if err == nil {
		return movie, nil
	}

	movie = &db.Movie{BaseItem: db.BaseItem{TmdbID: tmdbID}}
	if err := m.UpdateMovieMD(movie); err != nil {
		return nil, err
	}

	if m.Subscriber != nil {
		m.Subscriber.MovieAdded(movie)
	}

	return movie, nil
}
