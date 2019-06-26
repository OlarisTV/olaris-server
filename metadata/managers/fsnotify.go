package managers

import (
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/filesystem"
	"time"
)

func (man *LibraryManager) startWatcher(exitChan chan bool) {
	log.WithFields(log.Fields{"libraryID": man.Library.ID}).Println("Starting FSNotify watcher")
loop:
	for {
		select {
		case <-exitChan:
			log.WithFields(log.Fields{"libraryID": man.Library.ID}).Println("Stopping FSNotify watchers.")
			man.Watcher.Close()
			break loop
		case event := <-man.Watcher.Events:
			log.WithFields(log.Fields{"filename": event.Name, "event": event.Op}).Debugln("Got filesystem notification event.")

			if event.Op&fsnotify.Remove == fsnotify.Remove {
				man.Watcher.Remove(event.Name)
				man.CheckRemovedFiles() // Make this faster by only scanning the changed file
				return
			}

			n, err := filesystem.LocalNodeFromPath(event.Name)
			if err != nil {
				log.WithFields(log.Fields{"error": err}).Warnln("Error getting node from filesystem.")
				return
			}

			if n.IsDir() {
				man.AddWatcher(event.Name)
				man.Refresh()
			} else if ValidFile(n) {
				// We are sleeping 2 seconds here in case it's a creation event and the file is 0kb but growing.
				time.Sleep(2 * time.Second)

				if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Debugln("File is renamed, forcing removed files scan.")
					man.CheckRemovedFiles() // Make this faster by only scanning the changed file
				}

				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Debugln("File added was added adding watcher and requesting library rescan.")
					man.AddWatcher(event.Name)
					man.ProbeFile(n)
				}

			}
		case err := <-man.Watcher.Errors:
			log.Warnln("fsnotify watcher error:", err)
		}
	}
}
