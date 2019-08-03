package managers

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/ncw/rclone/vfs"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/db"
	mhelpers "gitlab.com/olaris/olaris-server/metadata/helpers"
	"gitlab.com/olaris/olaris-server/metadata/parsers"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

var seriesMutex = &sync.Mutex{}
var moviesMutex = &sync.Mutex{}

type probeJob struct {
	node filesystem.Node
	man  *LibraryManager
}

// LibraryManager manages all active libraries.
type LibraryManager struct {
	Watcher       *fsnotify.Watcher
	Pool          *WorkerPool
	Library       *db.Library
	exitChan      chan bool
	isShutingDown bool
}

// NewLibraryManager creates a new LibraryManager with a pool worker that can process episode information.
func NewLibraryManager(lib *db.Library, s LibrarySubscriber) *LibraryManager {
	var err error
	manager := LibraryManager{Pool: NewDefaultWorkerPool(), Library: lib, exitChan: make(chan bool)}
	manager.Pool.SetSubscriber(s)

	manager.Watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Errorln("Could not start fsnotify")
	} else {
	}
	go manager.startWatcher(manager.exitChan)
	log.WithFields(log.Fields{"libraryID": lib.ID}).Println("Created new LibraryManager")

	return &manager
}

// Shutdown shuts down the LibraryManager, right now it's just about cleaning up the fsnotify watcher.
func (man *LibraryManager) Shutdown() {
	log.WithFields(log.Fields{"libraryID": man.Library.ID}).Debugln("Closing down LibraryManager")
	man.isShutingDown = true
	man.exitChan <- true
	man.Pool.Shutdown()
}

// IdentifyUnidentifiedFiles looks for missing metadata information and attempts to retrieve it.
func (man *LibraryManager) IdentifyUnidentifiedFiles() {
	switch kind := man.Library.Kind; kind {
	case db.MediaTypeMovie:
		log.WithFields(man.Library.LogFields()).Println("Updating metadata for movies.")
		man.IdentifyUnidentifiedMovieFiles()
	case db.MediaTypeSeries:
		log.WithFields(man.Library.LogFields()).Println("Updating metadata for TV.")
		man.IdentifyUnidentifiedEpisodeFiles()
	}
}

// IdentifyUnidentifiedEpisodeFiles loops over all series with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) IdentifyUnidentifiedEpisodeFiles() error {
	episodeFiles, err := db.FindAllUnidentifiedEpisodeFilesInLibrary(man.Library.ID)
	if err != nil {
		return err
	}

	for _, episodeFile := range episodeFiles {
		_, err := GetOrCreateEpisodeForEpisodeFile(
			episodeFile, agents.NewTmdbAgent(), man.Pool.Subscriber)
		if err != nil {
			return err
		}
	}
	return nil
}

// ForceMovieMetadataUpdate refreshes all metadata for the given movies in a library, even if metadata already exists.
func (man *LibraryManager) ForceMovieMetadataUpdate() {
	agent := agents.NewTmdbAgent()
	for _, movie := range db.FindMoviesInLibrary(man.Library.ID) {
		UpdateMovieMD(&movie, agent)
	}
}

// ForceSeriesMetadataUpdate refreshes all data from the agent and updates the database record.
func (man *LibraryManager) ForceSeriesMetadataUpdate() {
	agent := agents.NewTmdbAgent()
	for _, series := range db.FindSeriesInLibrary(man.Library.ID) {
		UpdateSeriesMD(&series, agent)
		for _, season := range db.FindSeasonsForSeries(series.ID) {
			// Consider building a pool for this
			UpdateSeasonMD(&season, agent)
			for _, episode := range db.FindEpisodesForSeason(season.ID) {
				go func(episode *db.Episode) {
					defer checkPanic()
					man.Pool.tmdbPool.Process(episode)
				}(&episode)
			}
		}
	}
}

// IdentifyUnidentifiedMovieFiles loops over all movies with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) IdentifyUnidentifiedMovieFiles() error {
	movieFiles, err := db.FindAllUnidentifiedMovieFilesInLibrary(man.Library.ID)
	if err != nil {
		return err
	}

	for _, movieFile := range movieFiles {
		_, err := GetOrCreateMovieForMovieFile(
			movieFile, agents.NewTmdbAgent(), man.Pool.Subscriber)
		if err != nil {
			return err
		}
	}
	return nil
}

func (man *LibraryManager) checkAndAddProbeJob(node filesystem.Node) {
	library := man.Library
	if (library.Kind == db.MediaTypeSeries && !db.EpisodeFileExists(node.FileLocator().String())) ||
		(library.Kind == db.MediaTypeMovie && !db.MovieFileExists(node.FileLocator().String())) {

		// This is really annoying however when a tunny job is added to a closed pool it will throw a panic
		// Right now a job can still be running when we delete a library this recover catches the fact that the pool is closed but we are still queuing up
		// TODO: Somebody smarter than me figure out a better way of doing this
		go func(p *probeJob) {
			defer checkPanic()
			man.Pool.probePool.Process(p)
		}(&probeJob{man: man, node: node})
	} else {
		log.WithFields(log.Fields{"path": node.Path()}).
			Debugln("File already exists in library, not adding again.")
	}
}

// RescanFilesystem goes over the filesystem and parses filenames in the given library.
func (man *LibraryManager) RescanFilesystem() {
	log.WithFields(man.Library.LogFields()).Println("Scanning library for changed files.")
	stime := time.Now()

	// TODO: Move this into db package
	man.Library.RefreshStartedAt = stime
	man.Library.RefreshCompletedAt = time.Time{}
	db.SaveLibrary(man.Library)

	var rootNode filesystem.Node
	var err error

	// TODO: Should this be in it's own healthCheck method on the library or something?
	switch man.Library.Backend {
	case db.BackendLocal:
		rootNode, err = filesystem.LocalNodeFromPath(man.Library.FilePath)
	case db.BackendRclone:
		rootNode, err = filesystem.RcloneNodeFromPath(
			path.Join(man.Library.RcloneName, man.Library.FilePath))
	}

	if err != nil {
		log.
			WithFields(log.Fields{
				"backend":    man.Library.Backend,
				"rcloneName": man.Library.RcloneName,
				"path":       man.Library.FilePath,
				"error":      err.Error()}).
			Errorln("Failed to access library filesystem root node")
		man.Library.Healthy = false
		db.SaveLibrary(man.Library)
		return
	}

	man.Library.Healthy = true
	db.SaveLibrary(man.Library)

	// We don't need to handle the error here because we already handle it in walkFn
	_ = rootNode.Walk(func(walkPath string, n filesystem.Node, err error) error {
		if err != nil {
			log.WithFields(log.Fields{"error": err}).
				Warnf("Received an error while walking %s", walkPath)
		} else if ValidFile(n) {
			man.checkAndAddProbeJob(n)
		}
		// Watchers are only supported for the local backend
		if n.BackendType() == filesystem.BackendLocal {
			man.AddWatcher(walkPath)
		}

		return nil
	}, true)

	dur := time.Since(stime)
	log.Printf("Probing library '%s' took %f seconds", man.Library.FilePath, dur.Seconds())
	man.Library.RefreshCompletedAt = time.Now()
	db.SaveLibrary(man.Library)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warnln("Error while probing some files.")
		return
	}
}

// AddWatcher adds a fsnotify watcher to the given path.
func (man *LibraryManager) AddWatcher(filePath string) {
	log.WithFields(log.Fields{"filepath": filePath}).Debugln("Adding path to fsnotify.")

	// Since there is no way to get a list of current watchers we are just going to remove a watcher just in case.
	man.Watcher.Remove(filePath)

	err := man.Watcher.Add(filePath)
	if err != nil {
		log.Warnln("Could not add filesystem notification watcher:", err)
	}
}

// ProbeFile goes over the given file and tries to attempt to find out more information based on the filename.
func (man *LibraryManager) ProbeFile(n filesystem.Node) error {
	library := man.Library
	log.WithFields(log.Fields{"filepath": n.Path()}).Println("Parsing filepath.")

	basename := n.Name()

	switch kind := library.Kind; kind {
	case db.MediaTypeSeries:
		episodeFile := db.EpisodeFile{
			MediaItem: db.MediaItem{
				FileName:  basename,
				FilePath:  n.FileLocator().String(),
				Size:      n.Size(),
				LibraryID: library.ID,
			},
			Streams: collectStreams(n),
		}

		db.SaveEpisodeFile(&episodeFile)

		// Queue EpisodeFile to be associated with an Episode/Season/Series tree from TMDB
		go func(episodeFile *db.EpisodeFile) {
			defer checkPanic()
			man.Pool.tmdbPool.Process(episodeFile)
		}(&episodeFile)

	case db.MediaTypeMovie:
		mi := db.MediaItem{
			FileName:  basename,
			FilePath:  n.FileLocator().String(),
			Size:      n.Size(),
			LibraryID: library.ID,
		}

		movieFile := db.MovieFile{MediaItem: mi}
		movieFile.Streams = collectStreams(n)
		db.CreateMovieFile(&movieFile)

		// Queue MovieFile to be associated with a Movie from TMDB
		go func(movieFile *db.MovieFile) {
			defer checkPanic()
			man.Pool.tmdbPool.Process(movieFile)
		}(&movieFile)
	}
	return nil
}

// ValidFile checks whether the supplied filepath is a file that can be indexed by the metadata server.
func ValidFile(node filesystem.Node) bool {
	filePath := node.Name()
	if node.IsDir() {
		log.WithFields(log.Fields{"filepath": filePath}).Debugln("File is a directory, not scanning as file.")
		return false
	}

	if !SupportedExtensions[filepath.Ext(filePath)] {
		log.WithFields(log.Fields{"extension": filepath.Ext(filePath), "filepath": filePath}).Debugln("File is not a valid media file, file won't be indexed.")
		return false
	}

	// Ignore really small files
	if node.Size() < MinFileSize {
		log.WithFields(log.Fields{"size": node.Size(), "filepath": filePath}).
			Debugln("File is too small, file won't be indexed.")
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

	log.WithFields(log.Fields{"uuid": UUID}).
		Debugln("Looking to refresh metadata agent data.")
	movie, err := db.FindMovieByUUID(UUID)
	if err != nil {
		go mhelpers.WithLock(func() {
			UpdateMovieMD(movie, agents.NewTmdbAgent())
		}, movie.UUID)
		return true
	}

	series, err := db.FindSeriesByUUID(UUID)
	if err != nil {
		go mhelpers.WithLock(func() {
			UpdateSeriesMD(series, agents.NewTmdbAgent())
		}, series.UUID)
		return true
	}

	season, err := db.FindSeasonByUUID(UUID)
	if err != nil {
		go mhelpers.WithLock(func() {
			UpdateSeasonMD(season, agents.NewTmdbAgent())
		}, season.UUID)
		return true
	}

	episode, err := db.FindEpisodeByUUID(UUID)
	if err != nil {
		go mhelpers.WithLock(func() {
			UpdateEpisodeMD(episode, agents.NewTmdbAgent())
		}, episode.UUID)
		return true
	}
	return false
}

func GetOrCreateMovieForMovieFile(
	movieFile *db.MovieFile,
	agent agents.MetadataRetrievalAgent,
	subscriber LibrarySubscriber) (*db.Movie, error) {

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
	searchRes, err := agent.TmdbSearchMovie(parsedInfo.Title, options)
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
	firstResult := searchRes.Results[0] // Take the first result for now

	movie, err := GetOrCreateMovieByTmdbID(firstResult.ID, agent, subscriber)
	if err != nil {
		return nil, err
	}

	movieFile.Movie = *movie
	db.SaveMovieFile(movieFile)

	movie.MovieFiles = []db.MovieFile{*movieFile}
	return movie, nil
}

func GetOrCreateMovieByTmdbID(
	tmdbID int,
	agent agents.MetadataRetrievalAgent,
	subscriber LibrarySubscriber) (*db.Movie, error) {

	// Lock so that we don't create the same movie twice
	moviesMutex.Lock()
	defer moviesMutex.Unlock()

	movie, err := db.FindMovieByTmdbID(tmdbID)
	if err == nil {
		return movie, nil
	}

	movie = &db.Movie{BaseItem: db.BaseItem{TmdbID: tmdbID}}
	if err := UpdateMovieMD(movie, agent); err != nil {
		return nil, err
	}

	if subscriber != nil {
		subscriber.MovieAdded(movie)
	}

	return movie, nil
}

func GetOrCreateEpisodeForEpisodeFile(
	episodeFile *db.EpisodeFile,
	agent agents.MetadataRetrievalAgent,
	subscriber LibrarySubscriber) (*db.Episode, error) {

	if episodeFile.EpisodeID != 0 {
		return db.FindEpisodeByID(episodeFile.EpisodeID)
	}

	name := strings.TrimSuffix(episodeFile.FileName, filepath.Ext(episodeFile.FileName))
	parsedInfo := parsers.ParseSerieName(name)

	if parsedInfo.SeasonNum == 0 || parsedInfo.EpisodeNum == 0 {
		// We can't do anything if we don't know the season/episode number
		return nil, fmt.Errorf("Can't parse Season/Episode number from filename %s", name)
	}

	// Find a series for this Episode
	var options = make(map[string]string)
	if parsedInfo.Year != 0 {
		options["first_air_date_year"] = strconv.FormatUint(parsedInfo.Year, 10)
	}
	searchRes, err := agent.TmdbSearchTv(parsedInfo.Title, options)
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
	seriesInfo := searchRes.Results[0] // Take the first result for now

	episode, err := GetOrCreateEpisodeByTmdbID(
		seriesInfo.ID, parsedInfo.SeasonNum, parsedInfo.EpisodeNum,
		agent, subscriber)
	if err != nil {
		return nil, err
	}

	episodeFile.Episode = episode
	db.SaveEpisodeFile(episodeFile)

	episode.EpisodeFiles = []db.EpisodeFile{*episodeFile}

	return episode, nil
}

func GetOrCreateEpisodeByTmdbID(
	seriesTmdbID int, seasonNum int, episodeNum int,
	agent agents.MetadataRetrievalAgent,
	subscriber LibrarySubscriber) (*db.Episode, error) {

	// Lock so that we don't create the same episode twice
	seriesMutex.Lock()
	defer seriesMutex.Unlock()

	// TODO(Leon Handreke): Do this with a JOIN in the DB
	series, err := db.FindSeriesByTmdbID(seriesTmdbID)
	if err != nil {
		return nil, err
	}

	season, err := db.FindSeasonBySeasonNumber(series, seasonNum)
	if err != nil {
		return nil, err
	}

	episode, err := db.FindEpisodeByNumber(season, episodeNum)
	if err == nil {
		return episode, nil
	}

	episode = &db.Episode{Season: season, SeasonID: season.ID, EpisodeNum: episodeNum}
	if err := UpdateEpisodeMD(episode, agent); err != nil {
		return nil, err
	}

	if subscriber != nil {
		subscriber.EpisodeAdded(episode)
	}

	return episode, nil
}

func getOrCreateSeriesByTmdbID(
	seriesTmdbID int,
	agent agents.MetadataRetrievalAgent,
	subscriber LibrarySubscriber) (*db.Series, error) {

	// Lock so that we don't create the same series twice
	seriesMutex.Lock()
	defer seriesMutex.Unlock()

	series, err := db.FindSeriesByTmdbID(seriesTmdbID)
	if err == nil {
		return series, nil
	}

	series = &db.Series{BaseItem: db.BaseItem{TmdbID: seriesTmdbID}}
	if err := UpdateSeriesMD(series, agent); err != nil {
		return nil, err
	}

	if subscriber != nil {
		subscriber.SeriesAdded(series)
	}

	return series, nil
}

func getOrCreateSeasonByTmdbID(
	seriesTmdbID int, seasonNum int,
	agent agents.MetadataRetrievalAgent,
	subscriber LibrarySubscriber) (*db.Season, error) {

	// Lock so that we don't create the same series twice
	seriesMutex.Lock()
	defer seriesMutex.Unlock()

	series, err := getOrCreateSeriesByTmdbID(seriesTmdbID, agent, subscriber)
	if err != nil {
		return nil, err
	}

	season, err := db.FindSeasonBySeasonNumber(series, seasonNum)
	if err == nil {
		return season, nil
	}

	season = &db.Season{Series: series, SeriesID: series.ID, SeasonNumber: seasonNum}
	if err := UpdateSeasonMD(season, agent); err != nil {
		return nil, err
	}

	if subscriber != nil {
		subscriber.SeasonAdded(season)
	}

	return season, nil
}

// CheckFileAndDeleteIfMissing checks the given media file and if it's no longer present removes it from the database
func CheckFileAndDeleteIfMissing(m db.MediaFile) {
	log.WithFields(log.Fields{
		"path":    m.GetFilePath(),
		"library": m.GetLibrary().Name,
	}).Debugln("Checking to see if file still exists.")

	switch m.GetLibrary().Backend {
	case db.BackendLocal:
		p, err := filesystem.ParseFileLocator(m.GetFilePath())
		//		log.WithFields(log.Fields{"path": p.Path}).Debugln("Checking on local")
		_, err = filesystem.LocalNodeFromPath(p.Path)
		// TODO(Leon Handreke): Check if the error is actually not found
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Warnln("Received error while statting file")
			m.DeleteSelfAndMD()
		}
	case db.BackendRclone:
		p, err := filesystem.ParseFileLocator(m.GetFilePath())
		//		log.WithFields(log.Fields{"path": p.Path}).Debugln("Checking on Rclone")
		_, err = filesystem.RcloneNodeFromPath(p.Path)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Warnln("Received error while statting file")
			// We only delete on the file does not exist error. Any other errors are not enough reason to wipe the content.
			if err == vfs.ENOENT {
				m.DeleteSelfAndMD()
			}
		}
	}
}

// CheckRemovedFiles checks all files in the database to ensure they still exist, if not it attempts to remove the MD information from the db.
func (man *LibraryManager) CheckRemovedFiles() {
	log.WithFields(log.Fields{"libraryID": man.Library.ID}).Infoln("Checking for removed files.")

	for _, movieFile := range db.FindMovieFilesInLibrary(man.Library.ID) {
		CheckFileAndDeleteIfMissing(movieFile)
	}

	for _, file := range db.FindEpisodeFilesInLibrary(man.Library.ID) {
		CheckFileAndDeleteIfMissing(file)
	}
}

// RefreshAll rescans all files and attempts to find missing metadata information.
func (man *LibraryManager) RefreshAll() {
	man.CheckRemovedFiles()

	if man.Library.IsLocal() {
		man.AddWatcher(man.Library.FilePath)
	}

	man.RescanFilesystem()
	man.IdentifyUnidentifiedFiles()
}

// UpdateSeriesMD loops over all series with no tmdb information yet and attempts to retrieve the metadata.
func UpdateSeriesMD(series *db.Series, agent agents.MetadataRetrievalAgent) error {
	log.WithFields(log.Fields{"name": series.Name}).
		Println("Refreshing metadata for series.")
	agent.UpdateSeriesMD(series, series.TmdbID)
	db.SaveSeries(series)
	return nil
}

// UpdateEpisodeMD updates the database record with the latest data from the agent
func UpdateEpisodeMD(ep *db.Episode, agent agents.MetadataRetrievalAgent) error {
	agent.UpdateEpisodeMD(ep,
		ep.GetSeries().TmdbID, ep.GetSeason().SeasonNumber, ep.EpisodeNum)
	db.SaveEpisode(ep)
	return nil
}

// UpdateSeasonMD updates the database record with the latest data from the agent
func UpdateSeasonMD(season *db.Season, agent agents.MetadataRetrievalAgent) error {
	agent.UpdateSeasonMD(season, season.GetSeries().TmdbID, season.SeasonNumber)
	db.SaveSeason(season)
	return nil
}

// UpdateMovieMD updates the database record with the latest data from the agent
func UpdateMovieMD(movie *db.Movie, agent agents.MetadataRetrievalAgent) error {
	log.WithFields(log.Fields{"title": movie.Title}).Println("Refreshing metadata for movie.")

	if err := agent.UpdateMovieMetadata(movie); err != nil {
		return err
	}
	// TODO(Leon Handreke): return an error here.
	db.SaveMovie(movie)
	return nil
}

func checkPanic() {
	if r := recover(); r != nil {
		log.WithFields(log.Fields{"error": r}).Debugln("Recovered from panic in pool processing.")
	}
}

func collectStreams(n filesystem.Node) []db.Stream {
	log.WithFields(log.Fields{"filePath": n.FileLocator().String()}).
		Debugln("Reading stream information from file")
	var streams []db.Stream

	s, err := ffmpeg.GetStreams(n.FileLocator())
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Debugln("Received error while opening file for stream inspection")
		return streams
	}

	streams = append(streams, DatabaseStreamFromFfmpegStream(s.GetVideoStream()))

	for _, s := range s.AudioStreams {
		streams = append(streams, DatabaseStreamFromFfmpegStream(s))
	}

	for _, s := range s.SubtitleStreams {
		streams = append(streams, DatabaseStreamFromFfmpegStream(s))
	}

	return streams
}
