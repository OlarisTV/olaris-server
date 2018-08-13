package db

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/ryanbradynd05/go-tmdb"
	"gitlab.com/bytesized/bytesized-streaming/helpers"
	"path"
	"path/filepath"
	"strings"
)

type MetadataContext struct {
	Db             *gorm.DB
	Tmdb           *tmdb.TMDb
	Watcher        *fsnotify.Watcher
	LibraryManager *LibraryManager
	ExitChan       chan int
}

var env *MetadataContext

func NewDefaultMDContext() *MetadataContext {
	dbPath := helpers.MetadataConfigPath()
	return NewMDContext(dbPath, true)
}

func NewMDContext(dbPath string, dbLogMode bool) *MetadataContext {
	helpers.EnsurePath(dbPath)
	db, err := gorm.Open("sqlite3", path.Join(dbPath, "metadata.db"))
	db.LogMode(dbLogMode)
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s\n", err))
	}

	// Migrate the db-schema
	db.AutoMigrate(&Movie{}, &MovieFile{}, &Library{}, &Series{}, &Season{}, &Episode{}, &EpisodeFile{}, &User{}, &Invite{}, &PlayState{}, &Stream{})

	apiKey := "0cdacd9ab172ac6ff69c8d84b2c938a8"
	tmdb := tmdb.Init(apiKey)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(fmt.Sprintf("Could not start filesystem watcher: %s\n", err))
	}

	exitChan := make(chan int)

	env = &MetadataContext{Db: db, Tmdb: tmdb, Watcher: watcher, ExitChan: exitChan}

	libraryManager := NewLibraryManager(watcher)
	env.LibraryManager = libraryManager

	// Scan on start-up
	go libraryManager.RefreshAll()

	go env.StartWatcher(exitChan)

	return env
}

func (self *MetadataContext) StartWatcher(exitChan chan int) {
	fmt.Println("Starting FSNotify watcher")
loop:
	for {
		select {
		case <-exitChan:
			fmt.Println("Stopping watcher")
			self.Watcher.Close()
			break loop
		case event := <-self.Watcher.Events:
			//fmt.Println("event:", event)
			if supportedExtensions[filepath.Ext(event.Name)] {
				fmt.Println("Got filesystem notification for valid media file")
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					self.LibraryManager.CheckRemovedFiles() // Make this faster by only scanning the changed file
				}

				if event.Op&fsnotify.Remove == fsnotify.Remove {
					fmt.Println("File removed, removing watcher")
					self.Watcher.Remove(event.Name)
					self.LibraryManager.CheckRemovedFiles() // Make this faster by only scanning the changed file
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Println("modified file:", event.Name)
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					fmt.Println("Added file:", event.Name)
					self.Watcher.Add(event.Name)
					fmt.Println("asking lib to scan")
					for _, lib := range AllLibraries() {
						if strings.Contains(event.Name, lib.FilePath) {
							fmt.Println("Scanning file for lib:", lib.Name)
							self.LibraryManager.ProbeFile(&lib, event.Name)
							// We can probably only get the MD for the recently added file here
							self.LibraryManager.UpdateMD(&lib)
						}
					}
				}
			}
		case err := <-self.Watcher.Errors:
			fmt.Println("error:", err)
		}
	}
}
