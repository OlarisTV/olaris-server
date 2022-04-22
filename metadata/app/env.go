// Package app wraps all other important packages.
package app

import (
	"fmt"
	"math/rand"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers/metadata"
)

// MetadataContext is a container for all important vars.
type MetadataContext struct {
	Db      *gorm.DB
	Watcher *fsnotify.Watcher

	MetadataRetrievalAgent agents.MetadataRetrievalAgent
	MetadataManager        *metadata.MetadataManager

	// Currently unused
	ExitChan chan bool
}

// Cleanup cleans up any running threads / processes for the context.
func (m *MetadataContext) Cleanup() {
	// Currently unused
	// m.ExitChan <- true
	m.Db.Close()
	log.Infoln("closed all metadata context")
}

var env *MetadataContext

// NewDefaultMDContext creates a new env with sane defaults.
func NewDefaultMDContext() *MetadataContext {
	dbDir := viper.GetString("sqliteDir")
	helpers.EnsurePath(dbDir)

	dbPath := path.Join(dbDir, "metadata.db")
	return NewMDContext(db.DatabaseOptions{
		Connection: fmt.Sprintf("sqlite3://%s", dbPath),
		LogMode:    false,
	}, agents.NewTmdbAgent())
}

// NewTestingMDContext creates a new MetadataContext for testing
func NewTestingMDContext(agent agents.MetadataRetrievalAgent) *MetadataContext {
	var a agents.MetadataRetrievalAgent
	if agent == nil {
		a = agents.NewTmdbAgent()
	} else {
		a = agent
	}
	return NewMDContext(db.DatabaseOptions{
		Connection: db.InMemory,
		LogMode:    false,
	}, a)
}

// NewMDContext lets you create a more custom environment.
func NewMDContext(
	databaseOptions db.DatabaseOptions,
	agent agents.MetadataRetrievalAgent) *MetadataContext {
	rand.Seed(time.Now().UTC().UnixNano())

	helpers.InitLoggers(log.InfoLevel)

	log.Printf("olaris metadata server - version \"%s\"", helpers.Version)

	database := db.NewDb(databaseOptions)
	database.SetLogger(&GormLogger{})

	exitChan := make(chan bool)

	env = &MetadataContext{
		Db:                     database,
		ExitChan:               exitChan,
		MetadataRetrievalAgent: agent,
		MetadataManager:        metadata.NewMetadataManager(agent),
	}

	metadataRefreshTicker := time.NewTicker(15 * time.Second)
	go func() {
		for range metadataRefreshTicker.C {
			log.Debugln("Running periodic jobs")

			env.MetadataManager.RefreshAgentMetadataWithMissingArt()

			// Refresh running series to pick up episodes that might be airing next
			env.MetadataManager.RefreshRunningSeriesMetadata()

			metadataRefreshTicker.Reset(4 * time.Hour)
		}
	}()

	// This is just to be sure we don't have leftover metadata from programming errors
	// TODO(Leon Handreke): Have some reporting so that we can fix the bugs that lead to this and
	//  still reduce user pain.
	// TODO(Leon Handreke): Actually enable this, it breaks tests
	//go env.MetadataManager.GarbageCollectAllEpisodes()

	return env
}
