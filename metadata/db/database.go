// Package db handles database queries for the metadata server
package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gormigrate.v1"
	// Import sqlite dialect
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var db *gorm.DB

// InMemory can be passed as a database path to NewDb to create an in-memory database
const InMemory string = ":memory:"

// NewDb initializes a new database instance.
func NewDb(dbPath string, dbLogMode bool) *gorm.DB {
	var err error

	db, err = gorm.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=1000")
	db.LogMode(dbLogMode)
	//db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s\n", err))
	}

	// NOTE(Leon Handreke): We do this here because some databases were initialized and
	// used before we introduced gormigrate. This code can be removed once we are
	// sure that no users with v0.1.x databases remain.
	migrationsTableExists := 0
	db.Table("sqlite_master").
		Where("type = 'table'").
		Where("name = 'migrations'").
		Count(&migrationsTableExists)
	usersTableExists := 0
	db.Table("sqlite_master").
		Where("type = 'table'").
		Where("name = 'users'").
		Count(&usersTableExists)
	if migrationsTableExists == 0 && usersTableExists == 1 {
		db.Exec("CREATE TABLE migrations (id VARCHAR(255) PRIMARY KEY)")
		db.Exec("INSERT INTO migrations (id) VALUES ('SCHEMA_INIT')")
	}

	err = migrateSchema(db)
	if err != nil {
		log.Fatalf("Failed to migrate database: %s", err)
	}

	return db
}

var allModels = []interface{}{
	&Movie{}, &MovieFile{}, &Library{}, &Series{}, &Season{}, &Episode{},
	&EpisodeFile{}, &User{}, &Invite{}, &PlayState{}, &Stream{},
}

func initSchema(tx *gorm.DB) error {
	return db.AutoMigrate(allModels...).Error
}

func migrateSchema(db *gorm.DB) error {
	// Migrate the db-schema
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// you migrations here
		{
			// All our filepaths in the DB were migrated to "file locators" to support
			// the new rclone library types.
			ID: "2019-06-19-new-filepaths",
			Migrate: func(tx *gorm.DB) error {
				type MovieFile struct {
					gorm.Model
					FilePath string
				}
				var movieFiles []MovieFile
				db.Find(&movieFiles)
				for _, f := range movieFiles {
					f.FilePath = "local#" + f.FilePath
					db.Save(f)
				}

				type EpisodeFile struct {
					gorm.Model
					FilePath string
				}
				var episodeFiles []EpisodeFile
				db.Find(&episodeFiles)
				for _, f := range episodeFiles {
					f.FilePath = "local#" + f.FilePath
					db.Save(f)
				}

				return nil
			},
			Rollback: nil,
		},
		{
			// We now just have EpisodeFiles with no Episodes attached
			ID: "2019-08-03-remove-unidentified-episodes",
			Migrate: func(tx *gorm.DB) error {
				return db.Exec("DELETE FROM episodes WHERE tmdb_id = 0;").Error
			},
		}, {
			// We now just have EpisodeFiles with no Episodes attached
			ID: "2019-08-03-remove-unidentified-movies",
			Migrate: func(tx *gorm.DB) error {
				return db.Exec("DELETE FROM movies WHERE tmdb_id = 0;").Error
			},
		},
	})

	m.InitSchema(initSchema)
	err := m.Migrate()
	if err != nil {
		return err
	}

	// By default, just do the auto migrations automatically.
	return db.AutoMigrate(allModels...).Error
}
