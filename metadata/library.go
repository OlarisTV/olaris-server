package metadata

import (
	"github.com/jinzhu/gorm"
)

var supportedExtensions = map[string]bool{
	".mp4": true,
	".mkv": true,
	".mov": true,
	".avi": true,
}

type Library struct {
	gorm.Model
	Kind     MediaType
	FilePath string `gorm:"unique_index:idx_file_path"`
	Name     string
	Movies   []*movieResolver
	Episodes []*episodeResolver
}

/*
func NewLibrary(name string, mediatype MediaType, filepath string, db *gorm.DB, tmdb *tmdb.TMDb) *Library {
	library := Library{Kind: mediatype, Name: name, FilePath: filepath, db: db, tmdb: tmdb}
	return &library

}*/
