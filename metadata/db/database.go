// Package db handles database queries for the metadata server
package db

import (
	"fmt"
	"path"
	"strings"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/db/dialects/mysql"
	"gitlab.com/olaris/olaris-server/metadata/db/dialects/postgres"
	"gitlab.com/olaris/olaris-server/metadata/db/dialects/sqlite"
	"gopkg.in/gormigrate.v1"
)

var db *gorm.DB

const (
	SQLite      = "sqlite3"
	MySQL       = "mysql"
	PostgresSQL = "postgres"
	CockroachDB = "cockroachdb"
)

// DatabaseOptions holds information about how the database instance should be initialized
type DatabaseOptions struct {
	Connection string
	LogMode    bool
}

func getDefaultDbPath() (string, error) {
	dbDir := viper.GetString("server.sqliteDir")
	if err := helpers.EnsurePath(dbDir); err != nil {
		return "", err
	}

	return path.Join(dbDir, "metadata.db"), nil
}

func defaultDb(logMode bool) *gorm.DB {
	dbPath, err := getDefaultDbPath()
	if err != nil {
		panic(fmt.Sprintf("failed to get default database path: %s\n", err))
	}
	db, err = sqlite.NewSQLiteDatabase(dbPath, logMode)
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s\n", err))
	}

	log.WithField("path", dbPath).Println("using default (sqlite3) database")
	return db
}

// NewDb initializes a new database instance.
func NewDb(options DatabaseOptions) *gorm.DB {
	var err error

	databaseTokens := strings.Split(options.Connection, "://")
	if len(databaseTokens) == 0 {
		db = defaultDb(options.LogMode)
	} else if len(databaseTokens) == 2 {
		engine := databaseTokens[0]
		connection := databaseTokens[1]
		switch engine {
		case SQLite:
			db, err = sqlite.NewSQLiteDatabase(connection, options.LogMode)
			if err != nil {
				panic(fmt.Sprintf("failed to connect database: %s\n", err))
			}
			log.Println("using sqlite3 database driver")
		case MySQL:
			db, err = mysql.NewMySQLDatabase(connection, options.LogMode)
			if err != nil {
				panic(fmt.Sprintf("failed to connect database: %s\n", err))
			}
			log.Println("using MySQL database driver")
		case CockroachDB, PostgresSQL:
			// CockroachDB uses the Postgres driver
			// https://www.cockroachlabs.com/docs/stable/build-a-go-app-with-cockroachdb-gorm.html
			db, err = postgres.NewPostgresDatabase(connection, options.LogMode)
			if err != nil {
				panic(fmt.Sprintf("failed to connect database: %s\n", err))
			}
			log.Println("using postgres database driver")
		default:
			panic(fmt.Sprintf("unknown database engine: %s", engine))
		}
	} else {
		log.Debugf("unable to parse database connection string: %s, defaulting to sqlite3", options.Connection)
		db = defaultDb(options.LogMode)
	}

	err = migrateSchema(db)
	if err != nil {
		log.Fatalf("failed to migrate database: %s", err)
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
