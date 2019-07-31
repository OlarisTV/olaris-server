package agents_test

import (
	"github.com/stretchr/testify/assert"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

func TestSeasonLookup(t *testing.T) {
	season := db.Season{SeasonNumber: 1}
	series := &db.Series{BaseItem: db.BaseItem{TmdbID: 2426}, Seasons: []*db.Season{&season}}
	a := agents.NewTmdbAgent()

	a.UpdateSeasonMD(&season, series)

	assert.EqualValues(t, 7625, season.TmdbID)
}

func TestTmdbMovieLookup(t *testing.T) {
	movie := db.Movie{BaseItem: db.BaseItem{TmdbID: 76341}}
	a := agents.NewTmdbAgent()

	err := a.UpdateMovieMetadata(&movie)
	assert.NoError(t, err)

	assert.Equal(t, "Mad Max: Fury Road", movie.OriginalTitle)
	assert.EqualValues(t, 76341, movie.TmdbID)
}
