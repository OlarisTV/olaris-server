package managers

import (
	"github.com/Jeffail/tunny"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/parsers"
	"os"
	"path/filepath"
	"strings"
)

// MinFileSize defines how big a file has to be to be indexed.
const MinFileSize = 5e6 // 5MB

// SupportedExtensions is a list of all extensions that we will scan as valid mediafiles.
var SupportedExtensions = map[string]bool{
	".mp4":  true,
	".mkv":  true,
	".mov":  true,
	".avi":  true,
	".webm": true,
	".wmv":  true,
	".mpg":  true,
	".mpeg": true,
}

// LibraryManager manages all active libraries.
type LibraryManager struct {
	pool    *tunny.Pool
	watcher *fsnotify.Watcher
}

type episodePayload struct {
	series  db.Series
	season  db.Season
	episode db.Episode
}

// NewLibraryManager creates a new LibraryManager with a pool worker that can process episode information.
func NewLibraryManager(watcher *fsnotify.Watcher) *LibraryManager {
	manager := LibraryManager{}
	if watcher != nil {
		manager.watcher = watcher
	}
	// The MovieDB currently has a 40 requests per 10 seconds limit. Assuming every request takes a second then four workers is probably ideal.
	agent := agents.NewTmdbAgent()
	//TODO: We probably want a more global pool.
	manager.pool = tunny.NewFunc(4, func(payload interface{}) interface{} {
		log.Debugln("Spawning episode worker.")
		ep, ok := payload.(episodePayload)
		if ok {
			err := updateEpisodeMD(agent, &ep.episode, &ep.season, &ep.series)
			if err != nil {
				log.WithFields(log.Fields{"error": err}).Warnln("Got an error updating metadata for series.")
			} else {
				db.UpdateEpisode(&ep.episode)
			}
		}
		log.Debugln("Episode worker finished.")
		return nil
	})

	return &manager
}

// UpdateMD looks for missing metadata information and attempts to retrieve it.
func (man *LibraryManager) UpdateMD(library *db.Library) {
	switch kind := library.Kind; kind {
	case db.MediaTypeMovie:
		log.WithFields(library.LogFields()).Println("Updating metadata for movies.")
		man.UpdateMovieMD(library)
	case db.MediaTypeSeries:
		log.WithFields(library.LogFields()).Println("Updating metadata for TV.")
		man.UpdateSeriesMD(library)
	}
}

// UpdateEpisodeMD looks for missing episode metadata and attempts to retrieve it.
func (man *LibraryManager) UpdateEpisodeMD(tv db.Series, season db.Season, episode db.Episode) error {
	log.WithFields(log.Fields{"series": tv.Name, "episode": episode.EpisodeNum, "season": season.SeasonNumber}).Debugln("Attempting to find metadata.")
	return nil
}

// updateEpisodeMD supplies a global method to be used from the pool.
func updateEpisodeMD(a agents.EpisodeAgent, episode *db.Episode, season *db.Season, series *db.Series) error {
	return a.UpdateEpisodeMD(episode, season, series)
}

// UpdateEpisodesMD loops over all episode with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) UpdateEpisodesMD() error {
	episodes := db.FindAllUnidentifiedEpisodes()
	for i := range episodes {
		go func(episode *db.Episode) {
			season := db.FindSeason(episode.SeasonID)
			series := db.FindSerie(season.SeriesID)
			man.pool.Process(episodePayload{season: season, series: series, episode: *episode})
		}(&episodes[i])
	}
	return nil
}

func updateSeasonMD(a agents.SeasonAgent, season *db.Season, series *db.Series) error {
	return a.UpdateSeasonMD(season, series)
}

// UpdateSeasonMD loops over all seasons with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) UpdateSeasonMD() error {
	agent := agents.NewTmdbAgent()
	for _, season := range db.FindAllUnidentifiedSeasons() {
		series := db.FindSerie(season.SeriesID)
		updateSeasonMD(agent, &season, &series)
		db.UpdateSeason(&season)
	}
	return nil
}

func updateSeriesMD(a agents.SeriesAgent, series *db.Series) error {
	return a.UpdateSeriesMD(series)
}

// UpdateSeriesMD loops over all series with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) UpdateSeriesMD(library *db.Library) error {
	agent := agents.NewTmdbAgent()
	for _, series := range db.FindAllUnidentifiedSeries() {
		updateSeriesMD(agent, &series)
		db.UpdateSeries(&series)
	}

	man.UpdateSeasonMD()
	man.UpdateEpisodesMD()

	return nil
}

func updateMovieMD(mi agents.MovieAgent, movie *db.Movie) error {
	return mi.UpdateMovieMD(movie)
}

// UpdateMovieMD loops over all movies with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) UpdateMovieMD(library *db.Library) error {
	agent := agents.NewTmdbAgent()
	for _, movie := range db.FindAllUnidentifiedMovies() {
		log.WithFields(log.Fields{"title": movie.Title}).Println("Attempting to fetch metadata for unidentified movie.")
		updateMovieMD(agent, &movie)
		db.UpdateMovie(&movie)
	}
	return nil
}

// Probe goes over the filesystem and parses filenames in the given library.
func (man *LibraryManager) Probe(library *db.Library) {
	switch kind := library.Kind; kind {
	case db.MediaTypeMovie:
		log.WithFields(library.LogFields()).Println("Probing files for movie information.")
		man.ProbeMovies(library)
	case db.MediaTypeSeries:
		log.Println("Probing files for series information.")
		man.ProbeSeries(library)
	}
}

// ProbeSeries goes over the given library and attempts to get series information from filenames.
func (man *LibraryManager) ProbeSeries(library *db.Library) {
	err := filepath.Walk(library.FilePath, func(walkPath string, info os.FileInfo, err error) error {
		if ValidFile(walkPath) {
			man.AddWatcher(walkPath)
			man.AddWatcher(filepath.Dir(walkPath))

			if !db.EpisodeFileExists(walkPath) {
				man.ProbeFile(library, walkPath)
			}
		}
		return nil
	})
	if err != nil {
		log.Warnln("Error probing series:", err)
	}
}

// AddWatcher adds a fsnotify watcher to the given path.
func (man *LibraryManager) AddWatcher(filePath string) {
	err := man.watcher.Add(filePath)
	if err != nil {
		log.Warnln("FSNOTIFY FAILURE:", err)
	}
}

// ProbeFile goes over the given file and tries to attempt to find out more information based on the filename.
func (man *LibraryManager) ProbeFile(library *db.Library, filePath string) error {
	log.WithFields(log.Fields{"filepath": filePath}).Println("Parsing filepath.")
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		// This catches broken symlinks
		if _, ok := err.(*os.PathError); ok {
			log.WithFields(log.Fields{"error": err}).Warnln("Got an error while statting file.")
			return nil
		}
		return err
	}
	basename := fileInfo.Name()
	name := strings.TrimSuffix(basename, filepath.Ext(basename))

	switch kind := library.Kind; kind {
	case db.MediaTypeSeries:
		parsedInfo := parsers.ParseSerieName(name)
		if parsedInfo.SeasonNum != 0 && parsedInfo.EpisodeNum != 0 {
			mi := db.MediaItem{
				FileName:  basename,
				FilePath:  filePath,
				Size:      fileInfo.Size(),
				Title:     parsedInfo.Title,
				LibraryID: library.ID,
				Year:      parsedInfo.Year,
			}
			var series db.Series
			var season db.Season

			db.FirstOrCreateSeries(&series, db.Series{Name: parsedInfo.Title})
			newSeason := db.Season{SeriesID: series.ID, SeasonNumber: parsedInfo.SeasonNum}
			db.FirstOrCreateSeason(&season, newSeason)

			ep := db.Episode{SeasonNum: parsedInfo.SeasonNum, EpisodeNum: parsedInfo.EpisodeNum, SeasonID: season.ID}
			db.FirstOrCreateEpisode(&ep, ep)

			epFile := db.EpisodeFile{MediaItem: mi, EpisodeID: ep.ID}
			epFile.Streams = db.CollectStreams(filePath)

			// TODO(Maran) We might be adding double files in case it already exist
			db.UpdateEpisodeFile(&epFile)
		} else {
			log.WithFields(log.Fields{"title": parsedInfo.Title}).Warnln("Could not identify episode based on parsed filename.")
		}

	case db.MediaTypeMovie:
		mvi := parsers.ParseMovieName(name)
		// Create a movie stub so the metadata can get to work on it after probing
		movie := db.Movie{Title: mvi.Title, Year: mvi.Year}
		db.FirstOrCreateMovie(&movie, movie)

		mi := db.MediaItem{
			FileName:  basename,
			FilePath:  filePath,
			Size:      fileInfo.Size(),
			Title:     mvi.Title,
			Year:      mvi.Year,
			LibraryID: library.ID,
		}

		movieFile := db.MovieFile{MediaItem: mi, MovieID: movie.ID}
		movieFile.Streams = db.CollectStreams(filePath)
		db.CreateMovieFile(&movieFile)

	}
	return nil
}

// ValidFile checks whether the supplied filepath is a file that can be indexed by the metadata server.
func ValidFile(filePath string) bool {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "filepath": filePath}).Warnln("Got an error while getting file information, file won't be indexed.")
		return false
	}

	if fileInfo.IsDir() {
		log.WithFields(log.Fields{"error": err, "filepath": filePath}).Debugln("File is a directory, not scanning as file.")
		return false
	}

	if !SupportedExtensions[filepath.Ext(filePath)] {
		log.WithFields(log.Fields{"extension": filepath.Ext(filePath), "filepath": filePath}).Debugln("File is not a valid media file, file won't be indexed.")
		return false
	}

	// Ignore really small files
	if fileInfo.Size() < MinFileSize {
		log.WithFields(log.Fields{"size": fileInfo.Size(), "filepath": filePath}).Debugln("File is too small, file won't be indexed.")
		return false
	}

	return true
}

// ProbeMovies goes over the given library and attempts to get movie information from filenames.
func (man *LibraryManager) ProbeMovies(library *db.Library) {
	err := filepath.Walk(library.FilePath, func(walkPath string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}
		if ValidFile(walkPath) {
			man.AddWatcher(walkPath)
			man.AddWatcher(filepath.Dir(walkPath))

			if !db.MovieFileExists(walkPath) {
				man.ProbeFile(library, walkPath)
			} else {
				log.WithFields(log.Fields{"path": walkPath}).Println("File already exists in library, not adding again.")
			}
		}

		return nil
	})

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warnln("Error while probing some files.")
		return
	}
}

// CheckRemovedFiles checks all files in the database to ensure they still exist, if not it attempts to remove the MD information from the db.
func (man *LibraryManager) CheckRemovedFiles() {
	log.Infoln("Checking libraries to see if any files got removed since our last scan.")
	for _, movieFile := range db.FindAllMovieFiles() {
		log.WithFields(log.Fields{
			"path": movieFile.FilePath,
		}).Debugln("Checking to see if file still exists.")
		if !helpers.FileExists(movieFile.FilePath) {
			log.Debugln("Missing file, cleaning up MD", movieFile.FileName)
			movieFile.DeleteSelfAndMD()
		}
	}

	for _, file := range db.FindAllEpisodeFiles() {
		log.WithFields(log.Fields{
			"path": file.FilePath,
		}).Debugln("Checking to see if file still exists.")
		if !helpers.FileExists(file.FilePath) {
			log.Debugln("Missing file, cleaning up MD", file.FileName)
			file.DeleteSelfAndMD()
		}
	}
}

// RefreshAll rescans all files and attempts to find missing metadata information.
func (man *LibraryManager) RefreshAll() {
	for _, lib := range db.AllLibraries() {
		man.AddWatcher(lib.FilePath)

		man.CheckRemovedFiles()

		log.WithFields(lib.LogFields()).Println("Scanning library for changed files.")
		man.Probe(&lib)

		log.WithFields(lib.LogFields()).Infoln("Scanning library for metadata updates.")
		man.UpdateMD(&lib)

	}

	db.MergeDuplicateMovies()

	go db.CollectStreamKeyFrames()

	log.Println("Finished refreshing libraries.")
}
