// Package app wraps all other important packages.
package app

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/jasonlvhit/gocron"
	"github.com/jinzhu/gorm"
	"math/rand"
	"time"
	// Import sqlite dialect
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers"
)

// MetadataContext is a container for all important vars.
type MetadataContext struct {
	Db             *gorm.DB
	Watcher        *fsnotify.Watcher
	LibraryManager *managers.LibraryManager
	ExitChan       chan bool
}

// Cleanup cleans up any running threads / processes for the context.
func (m *MetadataContext) Cleanup() {
	m.ExitChan <- true
	m.Db.Close()
	log.Infoln("Closed all metadata context")
}

var env *MetadataContext

// NewDefaultMDContext creates a new env with sane defaults.
func NewDefaultMDContext() *MetadataContext {
	dbPath := helpers.MetadataConfigPath()
	return NewMDContext(dbPath, true, true)
}

// NewMDContext lets you create a more custom environment.
func NewMDContext(dbPath string, dbLogMode bool, verboseLog bool) *MetadataContext {
	rand.Seed(time.Now().UTC().UnixNano())

	logLevel := log.InfoLevel
	if verboseLog == true {
		logLevel = log.DebugLevel
	}

	helpers.InitLoggers(logLevel)

	log.Printf("Olaris Metadata Server - v%s", helpers.Version())

	db := db.NewDb(dbPath, dbLogMode)
	db.SetLogger(&GormLogger{})

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(fmt.Sprintf("Could not start filesystem watcher: %s\n", err))
	}

	exitChan := make(chan bool)

	env = &MetadataContext{Db: db, Watcher: watcher, ExitChan: exitChan}

	libraryManager := managers.NewLibraryManager(watcher)
	env.LibraryManager = libraryManager

	// Scan once on start-up
	go libraryManager.RefreshAll()

	go env.startWatcher(exitChan)

	gocron.Every(2).Hours().Do(managers.RefreshAgentMetadataWithMissingArt)
	gocron.Start()

	return env
}
