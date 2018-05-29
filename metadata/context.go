package metadata

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/ryanbradynd05/go-tmdb"
	"os/user"
	"path"
)

type MetadataContext struct {
	Db          *gorm.DB
	Tmdb        *tmdb.TMDb
	RefreshChan chan int
}

func NewMDContext() *MetadataContext {
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Failed to determine user's home directory: ", err.Error())
	}
	dbPath := path.Join(usr.HomeDir, ".config", "bss", "metadb")
	EnsurePath(dbPath)
	db, err := gorm.Open("sqlite3", path.Join(dbPath, "bsmdb_data.db"))
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s\n", err))
	}

	// Migrate the db-schema
	db.AutoMigrate(&MovieItem{}, &Library{})

	apiKey := "0cdacd9ab172ac6ff69c8d84b2c938a8"
	tmdb := tmdb.Init(apiKey)
	context := &MetadataContext{Db: db, Tmdb: tmdb}
	return context
}
