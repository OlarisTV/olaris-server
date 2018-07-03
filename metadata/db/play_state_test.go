package db

import "testing"

func createData() {
	ps := PlayState{Finished: true, Playtime: 13, UserID: 1}
	ps2 := PlayState{Finished: true, Playtime: 14, UserID: 1}

	series := TvSeries{Name: "Test"}
	episode := &TvEpisode{SeasonNum: 1, EpisodeNum: 1, Name: "Episode 1", PlayState: ps}
	episode2 := &TvEpisode{SeasonNum: 1, EpisodeNum: 2, Name: "Episode 2", PlayState: ps2}
	episode3 := &TvEpisode{SeasonNum: 1, EpisodeNum: 3, Name: "Episode 3"}
	episode4 := &TvEpisode{SeasonNum: 1, EpisodeNum: 4, Name: "Episode 4"}

	season := TvSeason{Name: "Season 1", TvEpisodes: []*TvEpisode{episode, episode2, episode3, episode4}}
	series.TvSeasons = []*TvSeason{&season}

	env.Db.Create(&series)
}

func TestAllPlayState(t *testing.T) {
	defer setupTest(t)()

	createData()

	pss := LatestPlayStates(1, 1)
	if len(pss) != 1 {
		t.Error("Expected one PlayState to return got", len(pss), "instead")
	}

	pss = LatestPlayStates(2, 1)
	if len(pss) != 2 {
		t.Error("Expected two PlayStates to return got", len(pss), "instead")
	}
}

func createSeries1() {
	series2 := TvSeries{Name: "Test 2"}
	ep := &TvEpisode{SeasonNum: 3, EpisodeNum: 3, Name: "Episode 3", PlayState: PlayState{Finished: false, Playtime: 33, UserID: 1}}
	ep2 := &TvEpisode{SeasonNum: 3, EpisodeNum: 4, Name: "Episode 4"}
	s := TvSeason{Name: "Season 3", TvEpisodes: []*TvEpisode{ep, ep2}}
	series2.TvSeasons = []*TvSeason{&s}
	env.Db.Create(&series2)
}
func TestContinueMovie(t *testing.T) {
	defer setupTest(t)()
	createMovieData()

	movies := UpNextMovies(1)
	if movies[0].Title != "Test" {
		t.Error("Got the wrong movie expected Test but got:", movies[0].Title)
	}
}
func TestContinuePlayResume(t *testing.T) {
	defer setupTest(t)()

	createSeries1()
	createData()

	episodes := UpNextEpisodes(1)
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

}
