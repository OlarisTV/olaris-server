// Package app wraps all other important packages.
package app

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers/metadata"
	"math/rand"
	"path"
	"time"
)

// MetadataContext is a container for all important vars.
type MetadataContext struct {
	Db      *gorm.DB
	Watcher *fsnotify.Watcher

	MetadataRetrievalAgent agents.MetadataRetrievalAgent
	MetadataManager        *metadata.MetadataManager

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
func NewDefaultMDContext() *MetadataContext {
	dbDir := helpers.MetadataConfigPath()
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
		Connection: fmt.Sprintf("sqlite3://%s", db.Memory),
		LogMode:    false,
	}, a)
}

// NewMDContext lets you create a more custom environment.
func NewMDContext(
	databaseOptions db.DatabaseOptions,
	agent agents.MetadataRetrievalAgent) *MetadataContext {
	rand.Seed(time.Now().UTC().UnixNano())

	helpers.InitLoggers(log.InfoLevel)

	log.Printf("Olaris Metadata Server - Version \"%s\"", helpers.Version)

	database := db.NewDb(databaseOptions)
	database.SetLogger(&GormLogger{})

	exitChan := make(chan bool)

	env = &MetadataContext{
		Db:                     database,
		ExitChan:               exitChan,
		MetadataRetrievalAgent: agent,
		MetadataManager:        metadata.NewMetadataManager(agent),
	}

	metadataRefreshTicker := time.NewTicker(2 * time.Hour)
	go func() {
		for range metadataRefreshTicker.C {
			env.MetadataManager.RefreshAgentMetadataWithMissingArt()
		}
	}()

	// This is just to be sure we don't have leftover metadata from programming errors
	// TODO(Leon Handreke): Have some reporting so that we can fix the bugs that lead to this and
	//  still reduce user pain.
	// TODO(Leon Handreke): Actually enable this, it breaks tests
	//go env.MetadataManager.GarbageCollectAllEpisodes()

	return env
}
