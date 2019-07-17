package db_test

import (
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

var movie db.Movie

func createMovieData() {
	mi := db.MediaItem{FilePath: "/tmp/test", Title: "Test.mkv"}
	stream := db.Stream{CodecName: "test"}
	mf := db.MovieFile{MediaItem: mi, Streams: []db.Stream{stream}}

	movie = db.Movie{
		Title:         "Test",
		OriginalTitle: "Mad Max: Road Fury",
		MovieFiles:    []db.MovieFile{mf},
	}
	db.CreateMovie(&movie)
	ps := db.PlayState{
		MediaUUID: movie.UUID,
		Finished:  false, Playtime: 33, UserID: 1}
	db.SavePlayState(&ps)
}

func setupTest(t *testing.T) func() {
	dbc := db.NewDb(db.InMemory, false)

	// Test teardown - return a closure for use by 'defer'
	return func() {
		// t is from the outer setupTest scope
		dbc.Close()
	}
}

func TestUUIDable(t *testing.T) {
	defer setupTest(t)()

	createMovieData()

	if movie.UUID == "" || movie.MovieFiles[0].UUID == "" {
		t.Errorf("Movie/File was created without a UUID\n")
	} else {
		t.Log("Movie UUID:", movie.UUID)
	}
}

func TestSearchMovieByTitle(t *testing.T) {
	defer setupTest(t)()
	createMovieData()
	var movies []db.Movie
	movies = db.SearchMovieByTitle(1, "max")
	if len(movies) == 0 {
		t.Error("Did not get any movies while searching")
		return
	}

	if movies[0].OriginalTitle != "Mad Max: Road Fury" {
		t.Error("Did not get the correct movie while searching")
	}
}

func TestCollectMovie(t *testing.T) {
	defer setupTest(t)()

	createMovieData()

	mov := db.FirstMovie()

	if len(mov.MovieFiles) != 0 {
		t.Error("Expected no movie files but got any still")
	}

	movies := []db.Movie{mov}
	db.CollectMovieInfo(movies, 1)
	t.Log(movies)
	if len(movies[0].MovieFiles) == 0 {
		t.Error("Expected movie to have files information after calling CollectMovieInfo but there was nothing present")
	}
	if len(movies[0].MovieFiles[0].Streams) == 0 {
		t.Error("Expected movie to have stream information after calling CollectMovieInfo but there was nothing present")
	}
}
