package managers

import (
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
	node filesystem.Node
	man  *LibraryManager
}

type episodePayload struct {
	series  db.Series
	season  db.Season
	episode db.Episode
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

// UpdateMD looks for missing metadata information and attempts to retrieve it.
func (man *LibraryManager) UpdateMD() {
	switch kind := man.Library.Kind; kind {
	case db.MediaTypeMovie:
		log.WithFields(man.Library.LogFields()).Println("Updating metadata for movies.")
		man.IdentifyUnidentMovies()
	case db.MediaTypeSeries:
		log.WithFields(man.Library.LogFields()).Println("Updating metadata for TV.")
		man.IdentifyUnidentSeries()
	}
}

// UpdateEpisodesMD loops over all episode with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) UpdateEpisodesMD() error {
	episodes := db.FindAllUnidentifiedEpisodes()
	for i := range episodes {
		func(episode *db.Episode) {
			season := db.FindSeason(episode.SeasonID)
			series := db.FindSerie(season.SeriesID)
			defer checkPanic()
			man.Pool.tmdbPool.Process(&episodePayload{season: season, series: series, episode: *episode})
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
	log.WithFields(log.Fields{"name": series.Name}).Println("Refreshing metadata for series.")
	agent := agents.NewTmdbAgent()
	agents.UpdateSeriesMD(agent, series)
	db.UpdateSeries(series)
	return nil
}

// IdentifyUnidentSeries loops over all series with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) IdentifyUnidentSeries() error {
	for _, series := range db.FindAllUnidentifiedSeries() {
		UpdateSeriesMD(&series)
	}

	man.UpdateSeasonMD()
	man.UpdateEpisodesMD()

	return nil
}

// UpdateMovieMD updates the database record with the latest data from the agent
func UpdateMovieMD(movie *db.Movie) error {
	log.WithFields(log.Fields{"title": movie.Title}).Println("Refreshing metadata for movie.")

	agent := agents.NewTmdbAgent()
	agents.UpdateMovieMD(agent, movie)
	db.UpdateMovie(movie)
	return nil
}

// ForceMovieMetadataUpdate refreshes all metadata for the given movies in a library, even if metadata already exists.
func (man *LibraryManager) ForceMovieMetadataUpdate() {
	for _, movie := range db.FindMoviesInLibrary(man.Library.ID, 0) {
		UpdateMovieMD(&movie)
	}
}

// ForceSeriesMetadataUpdate refreshes all data from the agent and updates the database record.
func (man *LibraryManager) ForceSeriesMetadataUpdate() {
	for _, series := range db.FindSeriesInLibrary(man.Library.ID) {
		UpdateSeriesMD(&series)
		for _, season := range db.FindSeasonsForSeries(series.ID) {
			// Consider building a pool for this
			UpdateSeasonMD(&season, &series)
			for _, ep := range db.FindEpisodesForSeason(season.ID, 1) {
				go func(p *episodePayload) {
					defer checkPanic()
					man.Pool.tmdbPool.Process(p)
				}(&episodePayload{season: season, series: series, episode: ep})
			}
		}
	}
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
func (man *LibraryManager) IdentifyUnidentMovies() error {
	for _, movie := range db.FindAllUnidentifiedMoviesInLibrary(man.Library.ID) {
		log.WithFields(log.Fields{"title": movie.Title}).Println("Attempting to fetch metadata for unidentified movie.")
		go func(m *db.Movie) {
			defer checkPanic()
			man.Pool.tmdbPool.Process(m)
		}(&movie)
	}
	return nil
}

func checkPanic() {
	if r := recover(); r != nil {
		log.WithFields(log.Fields{"error": r}).Debugln("Recovered from panic in pool processing.")
	}
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

// Refresh goes over the filesystem and parses filenames in the given library.
func (man *LibraryManager) Refresh() {
	log.WithFields(man.Library.LogFields()).Println("Scanning library for changed files.")
	stime := time.Now()

	// TODO: Move this into db package
	man.Library.RefreshStartedAt = stime
	man.Library.RefreshCompletedAt = time.Time{}
	db.UpdateLibrary(man.Library)

	var rootNode filesystem.Node
	var err error

	// TODO: Should this be in it's own healthCheck method on the library or something?
	if man.Library.Backend == db.BackendLocal {
		rootNode, err = filesystem.LocalNodeFromPath(man.Library.FilePath)
		if err != nil {
			log.WithFields(log.Fields{"path": man.Library.FilePath, "error": err.Error()}).Errorln("Got an error trying to create local rootnode")
			man.Library.Healthy = false
			db.UpdateLibrary(man.Library)
			return
		}
		man.Library.Healthy = true
		db.UpdateLibrary(man.Library)
	} else if man.Library.Backend == db.BackendRclone {
		rootNode, err = filesystem.RcloneNodeFromPath(path.Join(man.Library.RcloneName, man.Library.FilePath))
		if err != nil {
			log.WithFields(log.Fields{"rcloneName": man.Library.RcloneName, "error": err.Error()}).Errorln("Something went wrong when trying to connect to the Rclone remote")
			man.Library.Healthy = false
			db.UpdateLibrary(man.Library)
			return
		}
		man.Library.Healthy = true
		db.UpdateLibrary(man.Library)
	}

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
	db.UpdateLibrary(man.Library)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warnln("Error while probing some files.")
		return
	}
}

// AddWatcher adds a fsnotify watcher to the given path.
func (man *LibraryManager) AddWatcher(filePath string) {
	log.WithFields(log.Fields{"filepath": filePath}).Debugln("Adding path to fsnotify.")
	err := man.Watcher.Add(filePath)
	if err != nil {
		log.Warnln("Could not add filesystem notification watcher:", err)
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

// ProbeFile goes over the given file and tries to attempt to find out more information based on the filename.
func (man *LibraryManager) ProbeFile(n filesystem.Node) error {
	library := man.Library
	log.WithFields(log.Fields{"filepath": n.Path()}).Println("Parsing filepath.")

	basename := n.Name()
	name := strings.TrimSuffix(basename, filepath.Ext(basename))

	switch kind := library.Kind; kind {
	case db.MediaTypeSeries:
		parsedInfo := parsers.ParseSerieName(name)
		if parsedInfo.SeasonNum != 0 && parsedInfo.EpisodeNum != 0 {
			mi := db.MediaItem{
				FileName:  basename,
				FilePath:  n.FileLocator().String(),
				Size:      n.Size(),
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
			epFile.Streams = collectStreams(n)

			db.UpdateEpisodeFile(&epFile)

			go func(p *episodePayload) {
				defer checkPanic()
				man.Pool.tmdbPool.Process(p)
			}(&episodePayload{season: season, series: series, episode: ep})
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
			FilePath:  n.FileLocator().String(),
			Size:      n.Size(),
			Title:     mvi.Title,
			Year:      mvi.Year,
			LibraryID: library.ID,
		}

		movieFile := db.MovieFile{MediaItem: mi, MovieID: movie.ID}
		movieFile.Streams = collectStreams(n)
		db.CreateMovieFile(&movieFile)

		go func(movie *db.Movie) {
			defer checkPanic()
			man.Pool.tmdbPool.Process(movie)
		}(&movie)
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

	man.Refresh()
	man.UpdateMD()

	db.MergeDuplicateMovies()
}

// TODO: Please tell me we can do this in a cleaner way, this makes my eyes bleed :|

// FfmpegStreamFromDatabaseStream creates a ffmpeg stream object based on a database object
func FfmpegStreamFromDatabaseStream(s db.Stream) ffmpeg.Stream {
	return ffmpeg.Stream{
		StreamKey: ffmpeg.StreamKey{
			FileLocator: s.StreamKey.FileLocator,
			StreamId:    s.StreamKey.StreamId,
		},
		TotalDuration:    s.TotalDuration,
		TimeBase:         s.TimeBase,
		TotalDurationDts: ffmpeg.DtsTimestamp(s.TotalDurationDts),
		Codecs:           s.Codecs,
		CodecName:        s.CodecName,
		Profile:          s.Profile,
		BitRate:          s.BitRate,
		FrameRate:        s.FrameRate,
		Width:            s.Width,
		Height:           s.Height,
		StreamType:       s.StreamType,
		Language:         s.Language,
		Title:            s.Title,
		EnabledByDefault: s.EnabledByDefault,
	}
}

// DatabaseStreamFromFfmpegStream does the reverse of the above.
func DatabaseStreamFromFfmpegStream(s ffmpeg.Stream) db.Stream {
	return db.Stream{
		StreamKey: db.StreamKey{
			FileLocator: s.StreamKey.FileLocator,
			StreamId:    s.StreamKey.StreamId,
		},
		TotalDuration:    s.TotalDuration,
		TimeBase:         s.TimeBase,
		TotalDurationDts: int64(s.TotalDurationDts),
		Codecs:           s.Codecs,
		CodecName:        s.CodecName,
		Profile:          s.Profile,
		BitRate:          s.BitRate,
		FrameRate:        s.FrameRate,
		Width:            s.Width,
		Height:           s.Height,
		StreamType:       s.StreamType,
		Language:         s.Language,
		Title:            s.Title,
		EnabledByDefault: s.EnabledByDefault,
	}
}
