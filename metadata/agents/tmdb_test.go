package agents_test

import (
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

func TestSeasonLookup(t *testing.T) {
	season := db.Season{SeasonNumber: 1}
	series := &db.Series{BaseItem: db.BaseItem{TmdbID: 2426}, Seasons: []*db.Season{&season}}
	a := agents.NewTmdbAgent()
	a.UpdateSeasonMD(&season, series)

	if season.TmdbID != 7625 {
		t.Errorf("Expected tmdb-id to be set to '' but it was '%v' instead", season.TmdbID)
	}
}

func TestTmdbMovieLookup(t *testing.T) {
	movie := db.Movie{Title: "Max Max Road Fury", OriginalTitle: "", Year: 2015}
	a := agents.NewTmdbAgent()
	a.UpdateMovieMD(&movie)
	if movie.OriginalTitle != "Mad Max: Fury Road" {
		t.Errorf("Expected original title to be set to 'Max Max: Fury Road' but it was '%s' instead", movie.OriginalTitle)
	}

	if movie.TmdbID != 76341 {
		t.Errorf("Expected tmdb-id to be set to '' but it was '%v' instead", movie.TmdbID)
	}
}
