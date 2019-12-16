package managers

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// removeFileFromMapWithMutex cuts down on boilerplate by locking the
// mutex before removing the file from the map.
func removeFileFromMapWithMutex(path string, fileList map[string]chan fsnotify.Event, mutex *sync.RWMutex) {
	mutex.Lock()
	if c, exists := fileList[path]; exists {
		close(c)
		delete(fileList, path)
	}
	mutex.Unlock()
}

func (man *LibraryManager) startWatcher(exitChan chan bool) {
	log.WithFields(log.Fields{"libraryID": man.Library.ID}).Println("Starting FSNotify watcher")

	// delayMutex guards a map of filePath -> chan fsnotify.Event;
	// the channel for each file is used to communicate relevant
	// write events so that we don't try to import still-growing
	// media files.
	delayMutex := &sync.RWMutex{}
	possiblyGrowingFiles := map[string]chan fsnotify.Event{}

loop:
	for {
		select {
		case <-exitChan:
			log.WithFields(log.Fields{"libraryID": man.Library.ID}).Println("Stopping FSNotify watchers.")
			man.Watcher.Close()
			break loop

		case event := <-man.Watcher.Events:
			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				var n *filesystem.LocalNode
				var err error

				if n, err = filesystem.LocalNodeFromPath(event.Name); err != nil {
					log.WithError(err).Warnln("Error getting node from filesystem.")
					removeFileFromMapWithMutex(event.Name, possiblyGrowingFiles, delayMutex)
					break
				} else if n.IsDir() {
					man.RecursiveProbe(n)
					break
				}

				go func(filePath string) {
					log.WithField("file", filePath).Debugln("watching file import…")
					updateChannel := make(chan fsnotify.Event)

					delayMutex.Lock()
					possiblyGrowingFiles[filePath] = updateChannel
					delayMutex.Unlock()

					// starting now, schedule the probe for 5 seconds in the future
					// TODO: make user-configurable? maybe something like:
					// seconds := viper.GetInt("library.importSettleTime")
					timer := time.NewTimer(5 * time.Second)
					for {
						select {
						case <-updateChannel:
							// if new data has been written to this file, let's hit
							// the reset button on the timer. don't want to probe
							// the file before it's done copying
							timer.Reset(5 * time.Second)

							// we have to update n; otherwise, the size of 0 (from
							// the creation event) is cached
							if n, err = filesystem.LocalNodeFromPath(event.Name); err != nil {

								log.WithError(err).
									WithField("path", filePath).
									Debugf("could not update node")
								removeFileFromMapWithMutex(event.Name, possiblyGrowingFiles, delayMutex)
								return
							}
						case <-timer.C:
							// we haven't gotten a write event in at least 5 seconds;
							// assume the file is no longer being written and try to
							// schedule a probe
							log.WithField("file", filePath).
								Debugln("file has stabilized, probing")
							removeFileFromMapWithMutex(event.Name, possiblyGrowingFiles, delayMutex)

							if ValidFile(n) {
								man.checkAndAddProbeJob(n)
							}
							return
						}
					}
				}(event.Name)
			case event.Op&fsnotify.Write == fsnotify.Write:
				// don't log anything here; it just floods the log output
				// however, pass the event to the update channel for this file
				// to reset the timer before the file is probed
				delayMutex.RLock()
				updateChannel, exists := possiblyGrowingFiles[event.Name]
				delayMutex.RUnlock()
				if exists {
					updateChannel <- event
				} else {
					// if the user configures Trace-level logging, let them drink from the firehose
					log.WithField("path", event.Name).Trace("write event without seeing file's creation")
				}

			case event.Op&fsnotify.Remove == fsnotify.Remove:
				// handle remove & rename the same; we only get the old filename from the event anyway
				fallthrough

			case event.Op&fsnotify.Rename == fsnotify.Rename:
				log.WithField("path", event.Name).Debugf("got remove/rename event")
				man.Watcher.Remove(event.Name)
				removeFileFromMapWithMutex(event.Name, possiblyGrowingFiles, delayMutex)

				var n *filesystem.LocalNode
				var err error
				n, err = filesystem.LocalNodeFromPath(event.Name)
				if err != nil {
					log.WithError(err).Debugf("caught while handling removed file")
				}

				if movieFile, err := db.FindMovieFileByPath(n); err == nil {
					log.WithField("path", event.Name).Debugf("deleting movie")
					movieFile.DeleteSelfAndMD()
				} else if episodeFile, err := db.FindEpisodeFileByPath(n); err == nil {
					log.WithField("path", event.Name).Debugf("deleting episode")
					episodeFile.DeleteSelfAndMD()
				} else {
					// if there was no movie or episode in the database, no need to rescan
					continue
				}
			}

		case err := <-man.Watcher.Errors:
			log.Warnln("fsnotify watcher error:", err)
		}
	}
}
