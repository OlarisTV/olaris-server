package db

import (
	"testing"
)

func TestUUIDable(t *testing.T) {
	NewMDContext("/tmp/", false)
	movie := Movie{Title: "Test", MovieFiles: []MovieFile{MovieFile{FilePath: "/tmp/test", Title: "Test.mkv"}}}
	env.Db.Create(&movie)
	if movie.UUID == "" {
		t.Errorf("Stream was created without a UUID\n")
	} else {
		t.Log("Movie UUID:", movie.UUID)
	}
}
