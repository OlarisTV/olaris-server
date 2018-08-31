package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// LogFields defines some standard fields to include in logs.
func (lib *Library) LogFields() log.Fields {
	return log.Fields{"name": lib.Name, "path": lib.FilePath}
}

// Library is a struct containing information about filesystem folders.
type Library struct {
	gorm.Model
	Kind     MediaType
	FilePath string `gorm:"unique_index:idx_file_path"`
	Name     string
}

// AllLibraries returns all libraries from the database.
func AllLibraries() []Library {
	var libraries []Library
	db.Find(&libraries)
	return libraries
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
func AddLibrary(name string, filePath string, kind MediaType) (Library, error) {
	fmt.Printf("Add library '%s' with path '%s', type: '%d'\n", name, filePath, kind)
	lib := Library{Name: name, FilePath: filePath, Kind: kind}
	dbObj := db.Create(&lib)
	return lib, dbObj.Error
}
