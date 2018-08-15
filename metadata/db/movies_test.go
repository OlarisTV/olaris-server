package db

import (
	"fmt"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"io/ioutil"
	"os"
	"testing"
)

var movie Movie

func createMovieData() {
	mi := MediaItem{FilePath: "/tmp/test", Title: "Test.mkv"}
	stream := Stream{Stream: ffmpeg.Stream{CodecName: "test"}}
	mf := MovieFile{MediaItem: mi, Streams: []Stream{stream}}

	ps := PlayState{Finished: false, Playtime: 33, UserID: 1}

	movie = Movie{Title: "Test", OriginalTitle: "Mad Max: Road Fury", MovieFiles: []MovieFile{mf}, PlayState: ps}
	env.Db.Create(&movie)
}

func setupTest(t *testing.T) func() {
	tmp, err := ioutil.TempDir(os.TempDir(), "bss-test")
	if err == nil {
		fmt.Println("Creating DB in:", tmp)
		NewMDContext(tmp, true)
	} else {
		t.Error("Could not create tmpfile for database tests:", err)
	}

	// Test teardown - return a closure for use by 'defer'
	return func() {
		// t is from the outer setupTest scope
		env.Db.Close()
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
	var movies []Movie
	movies = SearchMovieByTitle(1, "max")
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

	var mov Movie
	env.Db.First(&mov)

	if len(mov.MovieFiles) != 0 {
		t.Error("Expected no movie files but got any still")
	}

	movies := []Movie{mov}
	CollectMovieInfo(movies, 1)
	t.Log(movies)
	if len(movies[0].MovieFiles) == 0 {
		t.Error("Expected movie to have files information after calling CollectMovieInfo but there was nothing present")
	}
	if len(movies[0].MovieFiles[0].Streams) == 0 {
		t.Error("Expected movie to have stream information after calling CollectMovieInfo but there was nothing present")
	}
	playstate := PlayState{}
	if movies[0].PlayState == playstate {
		t.Error("Expected movie to have Playstate information after calling CollectMovieInfo but there was nothing present")
	}
}
