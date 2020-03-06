package metadata

import (
	"fmt"
	"github.com/ryanbradynd05/go-tmdb"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/helpers/levenshtein"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/parsers"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

const xattrNameMovieTMDBID = "user.olaris.v1.movies.tmdb.id"

// ForceMovieMetadataUpdate refreshes all metadata for all movies
func (m *MetadataManager) ForceMovieMetadataUpdate() {
	for _, movie := range db.FindAllMovies(nil) {
		m.refreshAndSaveMovieMetadata(&movie)
	}
}

// refreshAndSaveMovieMetadata updates the database record with the latest data from the agent
func (m *MetadataManager) refreshAndSaveMovieMetadata(movie *db.Movie) error {
	if err := m.agent.UpdateMovieMD(movie, movie.TmdbID); err != nil {
		return err
	}
	log.WithFields(log.Fields{"title": movie.Title}).
		Println("Refreshed metadata for movie.")

	return db.SaveMovie(movie)
}

// Take a MovieFile object and try to read the TMDB ID from the extended file attributes
func (m *MetadataManager) getMovieTMDBIDFromXattr(
	movieFile *db.MovieFile) (tmdbID int, xattrInfoFound bool, err error) {

	// Need the file path
	p, err := filesystem.ParseFileLocator(movieFile.GetFilePath())
	if err != nil {
		return 0, false, err
	}

	xattrTmdbIDs, err := helpers.GetXattrInts(p.Path, []string{xattrNameMovieTMDBID})
	// TODO(Leon Handreke): Distinguish between fs read fail and no xattrs being found on the file
	if err != nil {
		log.WithFields(log.Fields{
			"filename": movieFile.GetFilePath(),
		}).Debugln("Failed to read xattrs")
		return 0, false, nil
	}

	return xattrTmdbIDs[xattrNameMovieTMDBID], true, nil
}

// Take a MovieFile object and try to determine the best
// TMDB ID by parsing the filename
func (m *MetadataManager) getMovieTMDBIDFromFilename(
	movieFile *db.MovieFile) (tmdbID int, found bool, err error) {
	name := strings.TrimSuffix(movieFile.FileName, filepath.Ext(movieFile.FileName))
	parsedInfo := parsers.ParseMovieName(name)

	var options = make(map[string]string)
	if parsedInfo.Year > 0 {
		options["year"] = strconv.FormatUint(parsedInfo.Year, 10)
	}

	searchRes, err := m.agent.TmdbSearchMovie(parsedInfo.Title, options)
	if err != nil {
		return 0, false, err
	}

	if len(searchRes.Results) == 0 {
		log.WithFields(log.Fields{
			"title": parsedInfo.Title,
			"year":  parsedInfo.Year,
		}).Warnln("Could not find match based on parsed title and given year.")

		return 0, false, nil
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

	return bestResult.ID, true, nil
}

func (m *MetadataManager) getMovieTMDBID(movieFile *db.MovieFile) (int, bool, error) {
	tmdbID, xattrInfoFound, err := m.getMovieTMDBIDFromXattr(movieFile)
	if err != nil {
		return 0, false, err
	}
	if xattrInfoFound {
		log.Debugln(
			"Read TMDB ID", tmdbID,
			"from xattr for", movieFile.FileName,
			"- skipping filename parse")
		return tmdbID, xattrInfoFound, nil
	}

	return m.getMovieTMDBIDFromFilename(movieFile)

}

// GetOrCreateMovieForMovieFile tries to create a Movie object by reading the TMDB ID stored
// in the filesystem extended attributes for the file, and then by parsing the filename of the
// given MovieFile and looking it up in TMDB. It associates the MovieFile with the new Model.
// If no matching movie can be found in TMDB, it returns an error.
func (m *MetadataManager) GetOrCreateMovieForMovieFile(
	movieFile *db.MovieFile) (*db.Movie, error) {

	// If we already have an associated movie, don't create a new one
	if movieFile.MovieID != 0 {
		return db.FindMovieByID(movieFile.MovieID)
	}

	// Nonstandard error handling logic here: the goal is to differentiate between
	// hitting an error when reading the xattr and merely not finding a match
	tmdbID, found, err := m.getMovieTMDBID(movieFile)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf(
			"Could not find match in TMDB for given filename: %s", movieFile.FileName)
	}

	movie, err := m.GetOrCreateMovieByTmdbID(tmdbID)
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
	if err := m.refreshAndSaveMovieMetadata(movie); err != nil {
		return nil, err
	}

	if m.Subscriber != nil {
		m.Subscriber.MovieAdded(movie)
	}

	return movie, nil
}
