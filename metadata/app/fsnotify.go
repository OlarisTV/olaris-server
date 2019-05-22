package app

import (
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers"
	"strings"
	"time"
)

func (env *MetadataContext) startWatcher(exitChan chan bool) {
	log.Println("Starting fsnotify watchers.")
loop:
	for {
		select {
		case <-exitChan:
			log.Println("Stopping fsnotify watchers.")
			env.Watcher.Close()
			break loop
		case event := <-env.Watcher.Events:
			log.WithFields(log.Fields{"filename": event.Name, "event": event.Op}).Debugln("Got filesystem notification event.")

			// We are sleeping 2 seconds here in case it's a creation event and the file is 0kb but growing.
			time.Sleep(2 * time.Second)

			if managers.IsDir(event.Name) {
				env.LibraryManager.AddWatcher(event.Name)
			} else if managers.ValidFile(event.Name) {
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Debugln("File is renamed, forcing removed files scan.")
					env.LibraryManager.CheckRemovedFiles() // Make this faster by only scanning the changed file
				}

				if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Debugln("File is removed, forcing removed files scan and removing fsnotify watch.")
					env.Watcher.Remove(event.Name)
					env.LibraryManager.CheckRemovedFiles() // Make this faster by only scanning the changed file
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Debugln("File added was added adding watcher and requesting library rescan.")
					env.Watcher.Add(event.Name)
					for _, lib := range db.AllLibraries() {
						if strings.Contains(event.Name, lib.FilePath) {
							env.LibraryManager.ProbeFile(&lib, event.Name)
							// We can probably only get the MD for the recently added file here
							env.LibraryManager.UpdateMD(&lib)
						}
					}
				}
			} else {
				log.WithFields(log.Fields{"filename": event.Name, "event": event.Op}).Debugln("Got an error while trying to open file. Going to assume the file was removed.")
				log.Debugln("File is removed, forcing removed files scan and removing fsnotify watch.")
				env.Watcher.Remove(event.Name)
				env.LibraryManager.CheckRemovedFiles() // Make this faster by only scanning the changed file
			}
		case err := <-env.Watcher.Errors:
			log.Warnln("fsnotify watcher error:", err)
		}
	}
}
