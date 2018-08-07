package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
)

type Library struct {
	gorm.Model
	Kind     MediaType
	FilePath string `gorm:"unique_index:idx_file_path"`
	Name     string
}

func AllLibraries() []Library {
	var libraries []Library
	env.Db.Find(&libraries)
	return libraries
}

func DeleteLibrary(id int) (Library, error) {
	library := Library{}
	env.Db.Find(&library, id)

	if library.ID != 0 {
		obj := env.Db.Unscoped().Delete(&library)
		if obj.Error == nil {
			if library.Kind == MediaTypeMovie {
				DeleteMoviesFromLibrary(library.ID)
			} else if library.Kind == MediaTypeSeries {
				DeleteEpisodesFromLibrary(library.ID)
			}
		}
		return library, obj.Error
	} else {
		return library, fmt.Errorf("Library not found, could not be deleted.")
	}
}

func AddLibrary(name string, filePath string, kind MediaType) (Library, error) {
	fmt.Printf("Add library '%s' with path '%s', type: '%d'\n", name, filePath, kind)
	lib := Library{Name: name, FilePath: filePath, Kind: kind}
	dbObj := env.Db.Create(&lib)
	return lib, dbObj.Error
}
