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
	ctx.Db.Find(&libraries)
	return libraries
}
func AddLibrary(name string, filePath string) {
	fmt.Printf("Add library '%s' with path '%s'", name, filePath)
	lib := Library{Name: name, FilePath: filePath}
	ctx.Db.Create(&lib)
}
