package db_test

import (
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

func createData() {
	ps := db.PlayState{Finished: true, Playtime: 13, UserID: 1}
	ps2 := db.PlayState{Finished: true, Playtime: 14, UserID: 1}

	series := db.Series{Name: "Test"}
	episode := &db.Episode{SeasonNum: 1, EpisodeNum: 1, Name: "Episode 1", PlayState: ps}
	episode2 := &db.Episode{SeasonNum: 1, EpisodeNum: 2, Name: "Episode 2", PlayState: ps2}
	episode3 := &db.Episode{SeasonNum: 1, EpisodeNum: 3, Name: "Episode 3"}
	episode4 := &db.Episode{SeasonNum: 1, EpisodeNum: 4, Name: "Episode 4"}

	season := db.Season{Name: "Season 1", Episodes: []*db.Episode{episode, episode2, episode3, episode4}}
	series.Seasons = []*db.Season{&season}
	db.CreateSeries(&series)
}

func TestAllPlayState(t *testing.T) {
	defer setupTest(t)()

	createData()

	pss := db.LatestPlayStates(1, 1)
	if len(pss) != 1 {
		t.Error("Expected one PlayState to return got", len(pss), "instead")
	}

	pss = db.LatestPlayStates(2, 1)
	if len(pss) != 2 {
		t.Error("Expected two PlayStates to return got", len(pss), "instead")
	}
}

func createSeries1() {
	series2 := db.Series{Name: "Test 2"}
	ep := &db.Episode{SeasonNum: 3, EpisodeNum: 3, Name: "Episode 3", PlayState: db.PlayState{Finished: false, Playtime: 33, UserID: 1}}
	ep2 := &db.Episode{SeasonNum: 3, EpisodeNum: 4, Name: "Episode 4"}
	s := db.Season{Name: "Season 3", Episodes: []*db.Episode{ep, ep2}}
	series2.Seasons = []*db.Season{&s}
	db.CreateSeries(&series2)
}
func TestContinueMovie(t *testing.T) {
	defer setupTest(t)()
	createMovieData()

	movies := db.UpNextMovies(1)
	if movies[0].Title != "Test" {
		t.Error("Got the wrong movie expected Test but got:", movies[0].Title)
	}
}
func TestContinuePlayResume(t *testing.T) {
	defer setupTest(t)()

	createSeries1()
	createData()

	episodes := db.UpNextEpisodes(1)
	t.Log("EPISODE:", episodes)
	if len(episodes) != 2 {
		t.Errorf("exepected %v episodes got %v instead", 2, len(episodes))
	}

	if episodes[0].Name != "Episode 3" {
		t.Errorf("Expected the first Episode to be resumed to be Episode 3 got %s instead\n", episodes[0].Name)
	}

	if episodes[1].Name != "Episode 3" {
		t.Errorf("Expected the second Episode to be resumed to be Episode 3 got %s instead\n", episodes[1].Name)
	}

	count := db.UnwatchedEpisodesInSeasonCount(1, 1)
	if count != 2 {
		t.Errorf("Expected the amount of unwatched episodes in the season to be 2 got %v instead\n", count)
	}

}
