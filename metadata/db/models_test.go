package db

import (
	"testing"
)

func TestUUIDable(t *testing.T) {
	NewMDContext("/tmp/", false)
	movie := Movie{Title: "Test", MovieFiles: []MovieFile{MovieFile{MediaItem: MediaItem{FilePath: "/tmp/test", Title: "Test.mkv"}}}}
	env.Db.Create(&movie)
	if movie.UUID == "" || movie.MovieFiles[0].UUID == "" {
		t.Errorf("Movie/File was created without a UUID\n")
	} else {
		t.Log("Movie UUID:", movie.UUID)
	}
}
