package managers

import (
	"github.com/fsnotify/fsnotify"
	"github.com/ncw/rclone/vfs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers/metadata"
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

// LibraryManager manages all active libraries.
type LibraryManager struct {
	metadataManager *metadata.MetadataManager
	Watcher         *fsnotify.Watcher
	Pool            *WorkerPool
	Library         *db.Library
	exitChan        chan bool
	isShuttingDown  bool
}

// NewLibraryManager creates a new LibraryManager
func NewLibraryManager(lib *db.Library, metadataManager *metadata.MetadataManager) *LibraryManager {
	var err error
	manager := LibraryManager{
		Library:         lib,
		metadataManager: metadataManager,
		Pool:            NewDefaultWorkerPool(),
		exitChan:        make(chan bool),
	}

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
	man.isShuttingDown = true
	man.exitChan <- true
	man.Pool.Shutdown()
}

// DeleteLibrary deletes the underlying Library object in the database and all associated files.
// Shutdown() must be called before calling this function! After that, the LibraryManager object
// must be discarded, it is no longer valid.
func (man *LibraryManager) DeleteLibrary() error {
	switch man.Library.Kind {
	case db.MediaTypeMovie:
		movieFiles, _ := db.FindMovieFilesInLibrary(man.Library.ID)
		for _, movieFile := range movieFiles {
			movieID := movieFile.MovieID
			movieFile.DeleteWithStreams()
			man.metadataManager.GarbageCollectMovieIfRequired(movieID)
		}
	case db.MediaTypeSeries:
		episodeFiles, _ := db.FindEpisodeFilesInLibrary(man.Library.ID)
		for _, episodeFile := range episodeFiles {
			episodeID := episodeFile.EpisodeID
			episodeFile.DeleteWithStreams()
			man.metadataManager.GarbageCollectEpisodeIfRequired(episodeID)
		}
	default:
		log.Error("Failed to delete library of kind", man.Library.Kind)
	}

	if err := db.DeleteLibraryByID(man.Library.ID); err != nil {
		return errors.Wrap(err, "Failed to delete library")
	}
	return nil
}

// IdentifyUnidentifiedFiles looks for missing metadata information and attempts to retrieve it.
func (man *LibraryManager) IdentifyUnidentifiedFiles() {
	log.WithFields(man.Library.LogFields()).
		Debugln("Trying to identify unidentified files in library.")
	var err error
	switch kind := man.Library.Kind; kind {
	case db.MediaTypeMovie:
		err = man.IdentifyUnidentifiedMovieFiles()
	case db.MediaTypeSeries:
		err = man.IdentifyUnidentifiedEpisodeFiles()
	}

	if err != nil {
		log.WithError(err).WithFields(man.Library.LogFields()).
			Warn("Failed to identify unidentified files in library")
	}
}

// IdentifyUnidentifiedEpisodeFiles loops over all series with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) IdentifyUnidentifiedEpisodeFiles() error {
	episodeFiles, err := db.FindAllUnidentifiedEpisodeFilesInLibrary(man.Library.ID)
	if err != nil {
		return err
	}

	for _, episodeFile := range episodeFiles {
		_, err := man.metadataManager.GetOrCreateEpisodeForEpisodeFile(episodeFile)
		if err != nil {
			log.WithError(err).WithField("episodeFile", episodeFile.FileName).
				Warn("Failed to identify EpisodeFile")
		}
	}
	return nil
}

// IdentifyUnidentifiedMovieFiles loops over all movies with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) IdentifyUnidentifiedMovieFiles() error {
	movieFiles, err := db.FindAllUnidentifiedMovieFilesInLibrary(man.Library.ID)
	if err != nil {
		return err
	}

	for _, movieFile := range movieFiles {
		_, err := man.metadataManager.GetOrCreateMovieForMovieFile(movieFile)
		if err != nil {
			log.WithError(err).WithField("movieFile", movieFile.FilePath).
				Warn("Failed to identify EpisodeFile")
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

// RescanFilesystem goes over the filesystem and parses filenames in the given library. If a filePath is supplied it will only scan the given path for new content.
func (man *LibraryManager) RescanFilesystem(filePath string) {
	if filePath == "" {
		filePath = man.Library.FilePath
		log.Debugln("No filePath supplied, going to scan from the root")
	} else {
		if strings.Contains(filePath, man.Library.FilePath) {
			log.WithFields(log.Fields{"filePath": filePath}).Debugln("Valid filepath supplied, scanning from giving path")
		} else {
			log.WithFields(log.Fields{"filePath": filePath, "libraryPath": man.Library.FilePath}).Debugln("Given filePath is not part of the library, ignoring.")
			filePath = man.Library.FilePath
		}
	}

	log.WithFields(man.Library.LogFields()).WithField("filePath", filePath).Println("Scanning library for changed files.")
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
		rootNode, err = filesystem.LocalNodeFromPath(filePath)
	case db.BackendRclone:
		rootNode, err = filesystem.RcloneNodeFromPath(
			path.Join(man.Library.RcloneName, filePath))
	}

	if err != nil {
		log.WithError(err).
			WithFields(log.Fields{
				"backend":    man.Library.Backend,
				"rcloneName": man.Library.RcloneName,
				"path":       man.Library.FilePath,
			}).
			Errorln("Failed to access library filesystem root node")
		man.Library.Healthy = false
		db.SaveLibrary(man.Library)
		return
	}

	man.Library.Healthy = true
	db.SaveLibrary(man.Library)

	man.RecursiveProbe(rootNode)

	dur := time.Since(stime)
	log.Printf("Scanning library took %f seconds", dur.Seconds())
	man.Library.RefreshCompletedAt = time.Now()
	db.SaveLibrary(man.Library)

	if err != nil {
		log.WithError(err).Warnln("error while probing files")
	}
}

// RecursiveProbe does what it says on the tin: recursively walks through a filesystem,
// starting from the given rootNode, adds watchers for all local subdirectories found,
// and probes any interesting files it finds along the way.
func (man *LibraryManager) RecursiveProbe(rootNode filesystem.Node) {
	log.WithField("path", rootNode.Path()).Debugf("RecursiveProbe called")

	if !strings.Contains(rootNode.Path(), man.Library.FilePath) {
		log.WithField("libraryRoot", man.Library.FilePath).
			Warnf("refusing to scan outside of library root")
		return
	}

	rootNode.Walk(func(walkPath string, n filesystem.Node, err error) error {
		p := filepath.Base(n.Path())
		if n.IsDir() && p[0] == '.' && viper.GetBool("metadata.scan_hidden") == false {
			log.WithFields(log.Fields{"path": p, "fullPath": n.Path()}).Warnln("skipping hidden folder, if you want to index it please set metadata.scan_hidden to true.")
			return filepath.SkipDir
		}

		if err != nil {
			log.WithError(err).Warnf("received an error while walking %s", walkPath)
		} else if ValidFile(n) {
			man.checkAndAddProbeJob(n)
		}

		// Watchers are only supported for the local backend
		if n.BackendType() == filesystem.BackendLocal && n.IsDir() {
			man.AddWatcher(n.FileLocator().Path)
		}

		return nil
	}, true)
}

// AddWatcher adds a fsnotify watcher to the given path.
func (man *LibraryManager) AddWatcher(filePath string) {
	log.WithFields(log.Fields{"filepath": filePath}).Debugln("Adding path to fsnotify.")

	// we always call man.Watcher.Add because it won't create redundant watchers if the filePath already exists.
	if err := man.Watcher.Add(filePath); err != nil {
		log.WithError(err).
			Warnln("could not add filesystem watcher; try increasing the sysctl fs.inotify.max_user_watches")
	}
}

// ProbeFile goes over the given file, creates a new entry in the database if required,
// and tries to associate the file with metadata based on the filename.
func (man *LibraryManager) ProbeFile(n filesystem.Node) error {
	library := man.Library
	log.WithFields(log.Fields{"filepath": n.Path()}).Println("Parsing filepath.")

	basename := n.Name()

	log.WithFields(log.Fields{"filePath": n.FileLocator().String()}).
		Debugln("Reading stream information from file")

	streams, err := ffmpeg.GetStreams(n.FileLocator())
	if err != nil {
		log.WithError(err).
			Debugln("Received error while opening file for stream inspection")
		return nil
	}

	// TODO(Leon Handreke): Ideally, to not have to scan the file at every startup,
	//  we would somehow create a database entry to remember that we already saw this file.
	// Ideally, this should happen in ValidFile,
	// but since we have to open and ffprobe the file, we do it in this async job instead.
	if len(streams.VideoStreams) == 0 {
		log.WithFields(log.Fields{"filePath": n.FileLocator().String()}).
			Infoln("File doesn't have any video streams, not adding to library.")
		return nil
	}

	switch kind := library.Kind; kind {
	case db.MediaTypeSeries:
		episodeFile := db.EpisodeFile{
			MediaItem: db.MediaItem{
				FileName:  basename,
				FilePath:  n.FileLocator().String(),
				Size:      n.Size(),
				LibraryID: library.ID,
			},
			Streams: collectStreams(streams),
		}

		db.SaveEpisodeFile(&episodeFile)

		_, err := man.metadataManager.GetOrCreateEpisodeForEpisodeFile(&episodeFile)
		if err != nil {
			log.WithError(err).WithField("episodeFile", episodeFile.FileName).
				Warn("Failed to to identify and create episode for EpisodeFile")
		}

	case db.MediaTypeMovie:
		movieFile := db.MovieFile{
			MediaItem: db.MediaItem{
				FileName:  basename,
				FilePath:  n.FileLocator().String(),
				Size:      n.Size(),
				LibraryID: library.ID,
			},
			Streams: collectStreams(streams),
		}
		db.SaveMovieFile(&movieFile)

		_, err := man.metadataManager.GetOrCreateMovieForMovieFile(&movieFile)
		if err != nil {
			log.WithError(err).WithField("movieFile", movieFile.FileName).
				Warn("Failed to to identify and create Movie for MovieFile")
		}

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

// FileMissing checks if the given media file is missing
func FileMissing(m db.MediaFile) bool {
	log.WithFields(log.Fields{
		"path":    m.GetFilePath(),
		"library": m.GetLibrary().Name,
	}).Debugln("Checking to see if file still exists.")

	switch m.GetLibrary().Backend {
	case db.BackendLocal:
		p, err := filesystem.ParseFileLocator(m.GetFilePath())
		if err != nil {
			log.WithError(err).Warnln("Received error while parsing local file locator")
			return true
		}
		_, err = filesystem.LocalNodeFromPath(p.Path)
		// TODO(Leon Handreke): Check if the error is actually not found
		if err != nil {
			log.WithError(err).Warnln("Received error while statting file")
			return true
		}
	case db.BackendRclone:
		p, err := filesystem.ParseFileLocator(m.GetFilePath())
		if err != nil {
			log.WithError(err).Warnln("Received error while parsing rclone file locator")
			return true
		}
		_, err = filesystem.RcloneNodeFromPath(p.Path)
		if err != nil {
			log.WithError(err).Warnln("Received error while statting file")
			// We only delete on the file does not exist error. Any other errors are not enough
			// reason to wipe the content.
			if err == vfs.ENOENT {
				return true
			}
		}
	}
	return false
}

// RemoveMissingFiles checks all files in the database to ensure they still exist;
// if not, it attempts to remove the MD information from the db.
func (man *LibraryManager) RemoveMissingFiles(locator filesystem.FileLocator) {
	log.WithFields(log.Fields{
		"libraryID": man.Library.ID,
		"locator":   locator,
	}).Infof("Checking for removed files under locator path")

	for _, movieFile := range db.FindMovieFilesInLibraryByLocator(man.Library.ID, locator) {
		if FileMissing(movieFile) {
			movieID := movieFile.MovieID
			movieFile.DeleteWithStreams()
			man.metadataManager.GarbageCollectMovieIfRequired(movieID)
		}
	}

	for _, episodeFile := range db.FindEpisodeFilesInLibraryByLocator(man.Library.ID, locator) {
		if FileMissing(episodeFile) {
			episodeID := episodeFile.EpisodeID
			episodeFile.DeleteWithStreams()
			man.metadataManager.GarbageCollectEpisodeIfRequired(episodeID)
		}
	}
}

// RefreshAll rescans all files and attempts to find missing metadata information.
func (man *LibraryManager) RefreshAll() {
	locator := filesystem.FileLocator{
		Backend: filesystem.BackendType(man.Library.Backend),
		Path:    man.Library.FilePath,
	}
	man.RemoveMissingFiles(locator)

	if man.Library.IsLocal() {
		man.AddWatcher(man.Library.FilePath)
	}

	man.RescanFilesystem("")
	man.IdentifyUnidentifiedFiles()
}

func checkPanic() {
	if r := recover(); r != nil {
		log.WithFields(log.Fields{"recover": r}).Debugln("Recovered from panic in pool processing.")
	}
}

func collectStreams(s *ffmpeg.Streams) []db.Stream {
	var streams []db.Stream

	for _, s := range s.VideoStreams {
		streams = append(streams, DatabaseStreamFromFfmpegStream(s))
	}

	for _, s := range s.AudioStreams {
		streams = append(streams, DatabaseStreamFromFfmpegStream(s))
	}

	for _, s := range s.SubtitleStreams {
		streams = append(streams, DatabaseStreamFromFfmpegStream(s))
	}

	return streams
}
