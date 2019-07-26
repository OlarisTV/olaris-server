// Package app wraps all other important packages.
package app

import (
	"github.com/fsnotify/fsnotify"
	"github.com/jinzhu/gorm"
	"math/rand"
	"path"
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
	Db       *gorm.DB
	Watcher  *fsnotify.Watcher
	ExitChan chan bool
}

// Cleanup cleans up any running threads / processes for the context.
func (m *MetadataContext) Cleanup() {
	m.ExitChan <- true
	m.Db.Close()
	log.Infoln("Closed all metadata context")
}

var env *MetadataContext

// NewDefaultMDContext creates a new env with sane defaults.
func NewDefaultMDContext(dbLogMode bool, verboseLog bool) *MetadataContext {
	dbDir := helpers.MetadataConfigPath()
	helpers.EnsurePath(dbDir)

	dbPath := path.Join(dbDir, "metadata.db")
	return NewMDContext(dbPath, dbLogMode, verboseLog)
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

	exitChan := make(chan bool)

	env = &MetadataContext{Db: db, ExitChan: exitChan}

	metadataRefreshTicker := time.NewTicker(2 * time.Hour)
	go func() {
		for range metadataRefreshTicker.C {
			managers.RefreshAgentMetadataWithMissingArt()
		}
	}()

	return env
}
