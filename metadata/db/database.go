// Package db handles database queries for the metadata server
package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"path"
	// Import sqlite dialect
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"gitlab.com/olaris/olaris-server/helpers"
)

var db *gorm.DB

// NewDb initializes a new database instance.
func NewDb(dbPath string, dbLogMode bool) *gorm.DB {
	var err error
	helpers.EnsurePath(dbPath)
	db, err = gorm.Open("sqlite3", path.Join(dbPath, "metadata.db"))
	db.LogMode(dbLogMode)
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s\n", err))
	}

	// Migrate the db-schema
	db.AutoMigrate(&Movie{}, &MovieFile{}, &Library{}, &Series{}, &Season{}, &Episode{}, &EpisodeFile{}, &User{}, &Invite{}, &PlayState{}, &Stream{})

	return db
}
