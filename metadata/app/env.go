// Package app wraps all other important packages.
package app

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/jinzhu/gorm"
	"math/rand"
	"time"
	// Import sqlite dialect
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers"
	"strings"
)

// MetadataContext is a container for all important vars.
type MetadataContext struct {
	Db             *gorm.DB
	Watcher        *fsnotify.Watcher
	LibraryManager *managers.LibraryManager
	ExitChan       chan int
}

var env *MetadataContext

// NewDefaultMDContext creates a new env with sane defaults.
func NewDefaultMDContext() *MetadataContext {
	dbPath := helpers.MetadataConfigPath()
	return NewMDContext(dbPath, true)
}

// NewMDContext lets you create a more custom environment.
func NewMDContext(dbPath string, dbLogMode bool) *MetadataContext {
	rand.Seed(time.Now().UTC().UnixNano())

	helpers.InitLoggers()
	log.Printf("Olaris-server - v%s", helpers.Version())

	db := db.NewDb(dbPath, dbLogMode)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(fmt.Sprintf("Could not start filesystem watcher: %s\n", err))
	}

	exitChan := make(chan int)

	env = &MetadataContext{Db: db, Watcher: watcher, ExitChan: exitChan}

	libraryManager := managers.NewLibraryManager(watcher)
	env.LibraryManager = libraryManager

	// Scan once on start-up
	go libraryManager.RefreshAll()

	go env.startWatcher(exitChan)

	return env
}

func (env *MetadataContext) startWatcher(exitChan chan int) {
	log.Println("Starting fsnotify watchers.")
loop:
	for {
		select {
		case <-exitChan:
			log.Println("Stopping fsnotify watchers.")
			env.Watcher.Close()
			break loop
		case event := <-env.Watcher.Events:
			if managers.ValidFile(event.Name) {
				log.WithFields(log.Fields{"filename": event.Name}).Debugln("Got filesystem notification for valid media file.")
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
					log.Debugln("File added was added adding watcher and requestin library rescan.")
					env.Watcher.Add(event.Name)
					for _, lib := range db.AllLibraries() {
						if strings.Contains(event.Name, lib.FilePath) {
							env.LibraryManager.ProbeFile(&lib, event.Name)
							// We can probably only get the MD for the recently added file here
							env.LibraryManager.UpdateMD(&lib)
						}
					}
				}
			}
		case err := <-env.Watcher.Errors:
			log.Warnln("fsnotify watcher error:", err)
		}
	}
}
