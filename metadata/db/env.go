package db

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/ryanbradynd05/go-tmdb"
	"gitlab.com/bytesized/bytesized-streaming/helpers"
	"path"
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

func NewMDContext() *MetadataContext {
	dbPath := path.Join(helpers.GetHome(), ".config", "bss", "metadb")
	helpers.EnsurePath(dbPath)
	db, err := gorm.Open("sqlite3", path.Join(dbPath, "bsmdb_data.db"))
	db.LogMode(true)
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s\n", err))
	}

	// Migrate the db-schema
	db.AutoMigrate(&Movie{}, &MovieFile{}, &Library{}, &TvSeries{}, &TvSeason{}, &TvEpisode{}, &EpisodeFile{}, &User{}, &Invite{}, &PlayState{})

	apiKey := "0cdacd9ab172ac6ff69c8d84b2c938a8"
	tmdb := tmdb.Init(apiKey)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(fmt.Sprintf("Could not start filesystem watcher: %s\n", err))
	}

	libraryManager := NewLibraryManager(watcher)
	// Scan on start-up
	go libraryManager.RefreshAll()

	exitChan := make(chan int)
	env = &MetadataContext{Db: db, Tmdb: tmdb, Watcher: watcher, ExitChan: exitChan, LibraryManager: libraryManager}
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
			fmt.Println("event:", event)
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				fmt.Println("File removed, removing watcher")
				self.Watcher.Remove(event.Name)
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
		case err := <-self.Watcher.Errors:
			fmt.Println("error:", err)
		}
	}
}
