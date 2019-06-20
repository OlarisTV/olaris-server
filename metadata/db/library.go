package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/helpers"
	"path"
	"time"
)

// LogFields defines some standard fields to include in logs.
func (lib *Library) LogFields() log.Fields {
	return log.Fields{"name": lib.Name, "path": lib.FilePath, "backend": lib.Backend}
}

// BackendType specifies what kind of Library backend is being used.
type BackendType int

const (
	// BackendLocal is used for local libraries
	BackendLocal = iota
	// BackendRclone is used for Rclone remotes
	BackendRclone
)

// Library is a struct containing information about filesystem folders.
type Library struct {
	gorm.Model
	Kind               MediaType
	Backend            int
	RcloneName         string
	FilePath           string `gorm:"unique_index:idx_file_path"`
	Name               string
	Healthy            bool `gorm:"default:'1'"`
	RefreshStartedAt   time.Time
	RefreshCompletedAt time.Time
}

// IsLocal returns true when a library is based on a local filesystem
func (lib *Library) IsLocal() bool {
	return lib.Backend == BackendLocal
}

// IsRclone returns true when a library is based on a rclone remote
func (lib *Library) IsRclone() bool {
	return lib.Backend == BackendRclone
}

// UpdateLibrary persists a library object in the database.
func UpdateLibrary(lib *Library) {
	db.Save(lib)
}

// AllLibraries returns all libraries from the database.
func AllLibraries() []Library {
	var libraries []Library
	db.Find(&libraries)
	return libraries
}

// FindLibrary finds a library.
func FindLibrary(id int) Library {
	var library Library
	db.Find(&library, id)
	return library
}

// DeleteLibrary deletes a library from the database.
func DeleteLibrary(id int) (Library, error) {
	library := Library{}
	db.Find(&library, id)

	if library.ID != 0 {
		obj := db.Unscoped().Delete(&library)
		if obj.Error == nil {
			if library.Kind == MediaTypeMovie {
				DeleteMoviesFromLibrary(library.ID)
			} else if library.Kind == MediaTypeSeries {
				DeleteEpisodesFromLibrary(library.ID)
			}
		}
		return library, obj.Error
	}

	return library, fmt.Errorf("library not found, could not be deleted")
}

// AddLibrary adds a filesystem folder and starts tracking media inside the folders.
func AddLibrary(lib *Library) error {
	if lib.Backend == BackendLocal && !helpers.FileExists(lib.FilePath) {
		return fmt.Errorf("supplied library path does not exist")
	}

	if lib.Backend == BackendRclone {
		if lib.RcloneName == "" {
			return fmt.Errorf("backend is set to rclone but no Rclone name has been specified")
		}

		_, err := filesystem.RcloneNodeFromPath(path.Join(lib.RcloneName, lib.FilePath))
		if err != nil {
			return fmt.Errorf("could not find path on rclone remote or remote threw an error: '%s'", err)
		}
	}

	log.WithFields(log.Fields{"name": lib.Name, "path": lib.FilePath, "kind": lib.Kind}).Infoln("Adding library")
	dbObj := db.Create(&lib)
	return dbObj.Error
}
