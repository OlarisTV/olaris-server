package managers

import (
	"github.com/Jeffail/tunny"
	"github.com/fsnotify/fsnotify"
	// Backends
	_ "github.com/ncw/rclone/backend/drive"
	_ "github.com/ncw/rclone/backend/local"
	"github.com/ncw/rclone/fs"
	"github.com/ncw/rclone/fs/walk"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/db"
	mhelpers "gitlab.com/olaris/olaris-server/metadata/helpers"
	"gitlab.com/olaris/olaris-server/metadata/parsers"
	"os"
	"path/filepath"
	"strings"
	"time"
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

type probeJob struct {
	fileStat FileStat
	library  *db.Library
}

// LibraryManager manages all active libraries.
type LibraryManager struct {
	pool    *tunny.Pool
	watcher *fsnotify.Watcher

	probeJobChan chan probeJob
}

type episodePayload struct {
	series  db.Series
	season  db.Season
	episode db.Episode
}

func (man *LibraryManager) probeFileWorker(id int) {
	for job := range man.probeJobChan {
		log.Debugf("Worker %d picked up job: %s.", id, job.fileStat.Name())
		man.ProbeFile(job.library, job.fileStat)
		log.Debugf("Worker %d done.", id)
	}
}

// NewLibraryManager creates a new LibraryManager with a pool worker that can process episode information.
func NewLibraryManager(watcher *fsnotify.Watcher) *LibraryManager {
	manager := LibraryManager{}

	if watcher != nil {
		manager.watcher = watcher
	}

	manager.probeJobChan = make(chan probeJob)

	for w := 1; w <= 4; w++ {
		go manager.probeFileWorker(w)
	}

	agent := agents.NewTmdbAgent()
	//TODO: We probably want a more global pool.
	// The MovieDB currently has a 40 requests per 10 seconds limit. Assuming every request takes a second then four workers is probably ideal.
	manager.pool = tunny.NewFunc(4, func(payload interface{}) interface{} {
		ep, ok := payload.(episodePayload)
		if ok {
			log.Debugln("Spawning episode worker.")
			err := agents.UpdateEpisodeMD(agent, &ep.episode, &ep.season, &ep.series)
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
		man.IdentifyUnidentMovies(library)
	case db.MediaTypeSeries:
		log.WithFields(library.LogFields()).Println("Updating metadata for TV.")
		man.IdentifyUnidentSeries(library)
	}
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

// UpdateSeasonMD loops over all seasons with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) UpdateSeasonMD() error {
	agent := agents.NewTmdbAgent()
	for _, season := range db.FindAllUnidentifiedSeasons() {
		series := db.FindSerie(season.SeriesID)
		agents.UpdateSeasonMD(agent, &season, &series)
		db.UpdateSeason(&season)
	}
	return nil
}

// UpdateSeriesMD loops over all series with no tmdb information yet and attempts to retrieve the metadata.
func UpdateSeriesMD(series *db.Series) error {
	agent := agents.NewTmdbAgent()
	agents.UpdateSeriesMD(agent, series)
	db.UpdateSeries(series)
	return nil
}

// IdentifyUnidentSeries loops over all series with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) IdentifyUnidentSeries(library *db.Library) error {
	for _, series := range db.FindAllUnidentifiedSeries() {
		UpdateSeriesMD(&series)
	}

	man.UpdateSeasonMD()
	man.UpdateEpisodesMD()

	return nil
}

// UpdateMovieMD updates the database record with the latest data from the agent
func UpdateMovieMD(movie *db.Movie) error {
	// Perhaps we should supply the agent to save to resources
	agent := agents.NewTmdbAgent()
	agents.UpdateMovieMD(agent, movie)
	db.UpdateMovie(movie)
	return nil
}

// RefreshAllMovieMD refreshes all data from the agent and updates the database record.
func RefreshAllMovieMD() error {
	for _, movie := range db.FindMoviesForMDRefresh() {
		log.WithFields(log.Fields{"title": movie.Title}).Println("Refreshing metadata for movie.")
		UpdateMovieMD(&movie)
	}
	return nil
}

// RefreshAllSeriesMD refreshes all data from the agent and updates the database record.
func RefreshAllSeriesMD() error {
	for _, series := range db.FindSeriesForMDRefresh() {
		log.WithFields(log.Fields{"name": series.Name}).Println("Refreshing metadata for series.")
		UpdateSeriesMD(&series)
		for _, season := range db.FindSeasonsForSeries(series.ID) {
			log.WithFields(log.Fields{"name": season.Name}).Println("Refreshing metadata for series.")
			UpdateSeasonMD(&season, &series)
			for _, ep := range db.FindEpisodesForSeason(season.ID, 1) {
				log.WithFields(log.Fields{"name": ep.Name}).Println("Refreshing metadata for episode.")
				UpdateEpisodeMD(&ep, &season, &series)
			}
		}
	}
	return nil
}

// UpdateEpisodeMD updates the database record with the latest data from the agent
func UpdateEpisodeMD(ep *db.Episode, season *db.Season, series *db.Series) error {
	agent := agents.NewTmdbAgent()
	agents.UpdateEpisodeMD(agent, ep, season, series)
	db.UpdateEpisode(ep)
	return nil
}

// UpdateSeasonMD updates the database record with the latest data from the agent
func UpdateSeasonMD(season *db.Season, series *db.Series) error {
	agent := agents.NewTmdbAgent()
	agents.UpdateSeasonMD(agent, season, series)
	db.UpdateSeason(season)
	return nil
}

// IdentifyUnidentMovies loops over all movies with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) IdentifyUnidentMovies(library *db.Library) error {
	for _, movie := range db.FindAllUnidentifiedMovies() {
		log.WithFields(log.Fields{"title": movie.Title}).Println("Attempting to fetch metadata for unidentified movie.")
		UpdateMovieMD(&movie)
	}
	return nil
}

// Probe goes over the filesystem and parses filenames in the given library.
func (man *LibraryManager) Probe(library *db.Library) {
	stime := time.Now()

	library.RefreshStartedAt = stime
	library.RefreshCompletedAt = time.Time{}
	db.UpdateLibrary(library)
	var err error

	if library.Backend == 0 {
		err = filepath.Walk(library.FilePath, func(walkPath string, info os.FileInfo, err error) error {
			fs, err := NewLocalFileStat(walkPath)
			if IsDir(walkPath) {
				man.AddWatcher(walkPath)
			} else if ValidFile(fs) {
				man.AddWatcher(walkPath)
				if (library.Kind == db.MediaTypeSeries && !db.EpisodeFileExists(walkPath)) || (library.Kind == db.MediaTypeMovie && !db.MovieFileExists(walkPath)) {
					man.probeJobChan <- probeJob{library: library, fileStat: fs}
				} else {
					log.WithFields(log.Fields{"path": walkPath}).Debugln("File already exists in library, not adding again.")
				}
			}
			return nil
		})
	} else if library.Backend == 1 {
		log.Println("Got Rclone backend")
		filesystem, err := fs.NewFs("gdrive:/Media")
		if err != nil {
			log.Println(err)
		} else {
			walk.Walk(filesystem, "", true, -1, func(walkPath string, entries fs.DirEntries, err error) error {
				log.Println("WALKPATH:", walkPath)
				for _, e := range entries {
					log.Println("ENTRY:", e)
					fs := &RcloneFileStat{dirEntry: e}
					if ValidFile(fs) {
						if (library.Kind == db.MediaTypeSeries && !db.EpisodeFileExists(walkPath)) || (library.Kind == db.MediaTypeMovie && !db.MovieFileExists(walkPath)) {
							man.probeJobChan <- probeJob{library: library, fileStat: fs}
						} else {
							log.WithFields(log.Fields{"path": walkPath}).Debugln("File already exists in library, not adding again.")
						}
					}
				}
				log.Println(err)
				return nil
			})
		}
	}

	dur := time.Since(stime)
	log.Printf("Probing library '%s' took %f seconds", library.FilePath, dur.Seconds())
	library.RefreshCompletedAt = time.Now()
	db.UpdateLibrary(library)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warnln("Error while probing some files.")
		return
	}
}

// AddWatcher adds a fsnotify watcher to the given path.
func (man *LibraryManager) AddWatcher(filePath string) {
	log.WithFields(log.Fields{"filepath": filePath}).Debugln("Adding path to fsnotify.")
	err := man.watcher.Add(filePath)
	if err != nil {
		log.Warnln("Could not add filesystem notification watcher:", err)
	}
}

// ProbeFile goes over the given file and tries to attempt to find out more information based on the filename.
func (man *LibraryManager) ProbeFile(library *db.Library, fileInfo FileStat) error {
	filePath := fileInfo.Path()
	log.WithFields(log.Fields{"filepath": filePath}).Println("Parsing filepath.")

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

			if series.TmdbID == 0 {
				log.Debugf("Series '%s' has no metadata yet, looking it up.", series.Name)
				UpdateSeriesMD(&series)
			}

			newSeason := db.Season{SeriesID: series.ID, SeasonNumber: parsedInfo.SeasonNum}
			db.FirstOrCreateSeason(&season, newSeason)
			if season.TmdbID == 0 {
				log.Debugf("Season %d for '%s' has no metadata yet, looking it up.", season.SeasonNumber, series.Name)
				UpdateSeasonMD(&season, &series)
			}

			ep := db.Episode{SeasonNum: parsedInfo.SeasonNum, EpisodeNum: parsedInfo.EpisodeNum, SeasonID: season.ID}
			db.FirstOrCreateEpisode(&ep, ep)

			epFile := db.EpisodeFile{MediaItem: mi, EpisodeID: ep.ID}
			epFile.Streams = db.CollectStreams(filePath)

			db.UpdateEpisodeFile(&epFile)

			UpdateEpisodeMD(&ep, &season, &series)
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
		if library.Backend == 1 {
			filePath = ("rclone://" + filePath)
		}
		movieFile.Streams = db.CollectStreams(filePath)
		db.CreateMovieFile(&movieFile)

		UpdateMovieMD(&movie)
	}
	return nil
}

// IsDir checks whether the given path is a directory.
// TODO: This should probably leave in a helper function
func IsDir(filePath string) bool {
	fileInfo, err := NewLocalFileStat(filePath)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "filepath": filePath}).Warnln("Got an error while statting file.")
		return false
	}
	if fileInfo.IsDir() {
		return true
	}

	return false
}

// ValidFile checks whether the supplied filepath is a file that can be indexed by the metadata server.
func ValidFile(fileInfo FileStat) bool {
	filePath := fileInfo.Name()
	if fileInfo.IsDir() {
		log.WithFields(log.Fields{"filepath": filePath}).Debugln("File is a directory, not scanning as file.")
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

// RefreshAgentMetadataWithMissingArt loops over all series/episodes/seasons and movies with missing art (posters/backdrop) and tries to retrieve them.
func RefreshAgentMetadataWithMissingArt() {
	log.Debugln("Checking and updating media items for missing art.")
	for _, UUID := range db.ItemsWithMissingMetadata() {
		RefreshAgentMetadataForUUID(UUID)
	}
}

// RefreshAgentMetadataForUUID takes an UUID of a mediaitem and refreshes all metadata
func RefreshAgentMetadataForUUID(UUID string) bool {
	log.WithFields(log.Fields{"uuid": UUID}).Debugln("Looking to refresh metadata agent data.")
	movies := db.FindMovieByUUID(&UUID, 0)
	if len(movies) > 0 {
		go mhelpers.WithLock(func() {
			UpdateMovieMD(&movies[0])
		}, movies[0].UUID)
		return true
	}

	series := db.FindSeriesByUUID(UUID)
	if len(series) > 0 {
		go mhelpers.WithLock(func() {
			UpdateSeriesMD(&series[0])
		}, series[0].UUID)
		return true
	}

	season := db.FindSeasonByUUID(UUID)
	if season.ID != 0 {
		go mhelpers.WithLock(func() {
			UpdateSeasonMD(&season, season.GetSeries())
		}, season.UUID)
		return true
	}

	episode := db.FindEpisodeByUUID(UUID, 0)
	if episode.ID != 0 {
		go mhelpers.WithLock(func() {
			UpdateEpisodeMD(&episode, episode.GetSeason(), episode.GetSeries())
		}, episode.UUID)
		return true
	}
	return false
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
	man.CheckRemovedFiles()

	for _, lib := range db.AllLibraries() {
		man.AddWatcher(lib.FilePath)

		log.WithFields(lib.LogFields()).Println("Scanning library for changed files.")
		man.Probe(&lib)

		log.WithFields(lib.LogFields()).Infoln("Scanning library for unidentified media.")
		man.UpdateMD(&lib)

	}

	RefreshAgentMetadataWithMissingArt()

	db.MergeDuplicateMovies()

	log.Println("Finished refreshing libraries.")
}
