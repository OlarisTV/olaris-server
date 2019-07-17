package db_test

import (
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

func createData() {
	series := db.Series{Name: "All Episodes completely watched"}
	episode := &db.Episode{SeasonNum: 1, EpisodeNum: 1, Name: "AECW - Episode 1"}
	db.UpdateEpisode(episode)
	db.SavePlayState(&db.PlayState{
		MediaUUID: episode.UUID,
		UserID:    1,
		Finished:  true, Playtime: 13,
	})

	episode2 := &db.Episode{SeasonNum: 1, EpisodeNum: 2, Name: "AECW - Episode 2"}
	db.UpdateEpisode(episode2)
	db.SavePlayState(&db.PlayState{
		MediaUUID: episode2.UUID,
		UserID:    1,
		Finished:  true, Playtime: 14,
	})
	episode3 := &db.Episode{SeasonNum: 1, EpisodeNum: 3, Name: "AECW - Episode 3"}
	episode4 := &db.Episode{SeasonNum: 1, EpisodeNum: 4, Name: "AECW - Episode 4"}

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
	series2 := db.Series{Name: "Not finished watching an episode yet"}
	ep := &db.Episode{SeasonNum: 3, EpisodeNum: 3, Name: "NFY - Episode 3"}
	db.UpdateEpisode(ep)
	db.SavePlayState(&db.PlayState{
		MediaUUID: ep.UUID,
		UserID:    1,
		Finished:  false, Playtime: 33,
	})

	ep2 := &db.Episode{SeasonNum: 3, EpisodeNum: 4, Name: "NFY - Episode 4"}
	s := db.Season{Name: "Season 3", Episodes: []*db.Episode{ep, ep2}}
	series2.Seasons = []*db.Season{&s}
	db.CreateSeries(&series2)
}
func createSeries2() {
	series := db.Series{Name: "Next Season"}
	episode := &db.Episode{SeasonNum: 1, EpisodeNum: 1, Name: "NS - Episode 1"}
	db.UpdateEpisode(episode)
	db.SavePlayState(&db.PlayState{
		MediaUUID: episode.UUID,
		UserID:    1,
		Finished:  true, Playtime: 13,
	})
	episode2 := &db.Episode{SeasonNum: 1, EpisodeNum: 2, Name: "NS - Episode 2"}
	db.UpdateEpisode(episode2)
	db.SavePlayState(&db.PlayState{
		MediaUUID: episode2.UUID,
		UserID:    1,
		Finished:  true, Playtime: 14,
	})

	episode3 := &db.Episode{SeasonNum: 2, EpisodeNum: 1, Name: "NS - Episode S02E01"}
	episode4 := &db.Episode{SeasonNum: 2, EpisodeNum: 2, Name: "NS - Episode S02E02"}

	season := db.Season{Name: "Season 1", Episodes: []*db.Episode{episode, episode2}}
	season2 := db.Season{Name: "Season 2", Episodes: []*db.Episode{episode4, episode3}}
	series.Seasons = []*db.Season{&season, &season2}
	db.CreateSeries(&series)
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
	createSeries2()
	createData()

	episodes := db.UpNextEpisodes(1)
	if len(episodes) != 3 {
		t.Errorf("exepected %v episodes got %v instead", 3, len(episodes))
	}

	if episodes[0].Name != "NFY - Episode 3" {
		t.Errorf("Expected the first Episode to be resumed to be %s got %s instead\n", "NFY - Episode 3", episodes[0].Name)
	}

	if episodes[1].Name != "NS - Episode S02E01" {
		t.Errorf("Expected the second Episode to be resumed to be NS - Episode S02E01 got %s instead\n", episodes[1].Name)
	}

	if episodes[2].Name != "AECW - Episode 3" {
		t.Errorf("Expected the second Episode to be resumed to be Episode 3 got %s instead\n", episodes[2].Name)
	}

	count := db.UnwatchedEpisodesInSeasonCount(1, 1)
	if count != 2 {
		t.Errorf("Expected the amount of unwatched episodes in the season to be 2 got %v instead\n", count)
	}

}
