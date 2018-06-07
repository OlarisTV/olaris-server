package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/ryanbradynd05/go-tmdb"
	"gitlab.com/bytesized/bytesized-streaming/helpers"
	"path"
)

type MetadataContext struct {
	Db   *gorm.DB
	Tmdb *tmdb.TMDb
}

var ctx *MetadataContext

func NewMDContext() *MetadataContext {
	dbPath := path.Join(helpers.GetHome(), ".config", "bss", "metadb")
	helpers.EnsurePath(dbPath)
	db, err := gorm.Open("sqlite3", path.Join(dbPath, "bsmdb_data.db"))
	db.LogMode(true)
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s\n", err))
	}

	// Migrate the db-schema
	db.AutoMigrate(&MovieItem{}, &Library{}, &TvSeries{}, &TvSeason{}, &TvEpisode{}, &User{})

	apiKey := "0cdacd9ab172ac6ff69c8d84b2c938a8"
	tmdb := tmdb.Init(apiKey)

	ctx = &MetadataContext{Db: db, Tmdb: tmdb}
	return ctx
}
