package managers

import (
	"github.com/fsnotify/fsnotify"
	"github.com/ncw/rclone/vfs"
	log "github.com/sirupsen/logrus"
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
			log.WithError(err).WithField("episodeFile", episodeFile).
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
	log.Printf("Probing library '%s' took %f seconds", man.Library.FilePath, dur.Seconds())
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
			log.WithError(err).WithField("episodeFile", episodeFile).
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
		db.CreateMovieFile(&movieFile)

		_, err := man.metadataManager.GetOrCreateMovieForMovieFile(&movieFile)
		if err != nil {
			log.WithError(err).WithField("movieFile", movieFile).
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
			log.WithError(err).Warnln("Received error while statting file")
			m.DeleteSelfAndMD()
		}
	case db.BackendRclone:
		p, err := filesystem.ParseFileLocator(m.GetFilePath())
		//		log.WithFields(log.Fields{"path": p.Path}).Debugln("Checking on Rclone")
		_, err = filesystem.RcloneNodeFromPath(p.Path)
		if err != nil {
			log.WithError(err).Warnln("Received error while statting file")
			// We only delete on the file does not exist error. Any other errors are not enough reason to wipe the content.
			if err == vfs.ENOENT {
				m.DeleteSelfAndMD()
			}
		}
	}
}

// CheckRemovedFiles checks all files in the database to ensure they still exist;
// if not, it attempts to remove the MD information from the db.
func (man *LibraryManager) CheckRemovedFiles(locator filesystem.FileLocator) {
	log.WithFields(log.Fields{
		"libraryID": man.Library.ID,
		"locator":   locator,
	}).Infof("Checking for removed files under locator path")

	for _, movieFile := range db.FindMovieFilesInLibraryByLocator(man.Library.ID, locator) {
		CheckFileAndDeleteIfMissing(movieFile)
	}

	for _, episodeFile := range db.FindEpisodeFilesInLibraryByLocator(man.Library.ID, locator) {
		CheckFileAndDeleteIfMissing(episodeFile)
	}
}

// RefreshAll rescans all files and attempts to find missing metadata information.
func (man *LibraryManager) RefreshAll() {
	locator := filesystem.FileLocator{
		Backend: filesystem.BackendType(man.Library.Backend),
		Path:    man.Library.FilePath,
	}
	man.CheckRemovedFiles(locator)

	if man.Library.IsLocal() {
		man.AddWatcher(man.Library.FilePath)
	}

	man.RescanFilesystem()
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
