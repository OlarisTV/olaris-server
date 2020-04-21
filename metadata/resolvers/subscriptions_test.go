package resolvers

import (
	"context"
	"github.com/stretchr/testify/assert"
	"gitlab.com/olaris/olaris-server/metadata/agents/agentsfakes"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
	"time"
)

func TestResolver_MoviesChanged_CreateMovie(t *testing.T) {
	tmdbAgent := agentsfakes.FakeMetadataRetrievalAgent{}
	tmdbAgent.UpdateMovieMDStub = func(movie *db.Movie, tmdbID int) error {
		movie.TmdbID = tmdbID
		movie.Title = "North of the Sun"
		return nil
	}

	metadataCtx := app.NewTestingMDContext(&tmdbAgent)
	r := NewResolver(metadataCtx)

	subCh := r.MoviesChanged(context.Background())

	metadataCtx.MetadataManager.GetOrCreateMovieByTmdbID(1234)

	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Timeout waiting for MovieAddedEvent")
	case e := <-subCh:
		eventResolver, ok := e.ToMovieAddedEvent()
		assert.True(t, ok)
		if ok {
			assert.NotNil(t, eventResolver.Movie())
		}
	}
}

func TestResolver_MoviesChanged_RefreshMovieMetadata(t *testing.T) {
	tmdbAgent := agentsfakes.FakeMetadataRetrievalAgent{}
	tmdbAgent.UpdateMovieMDStub = func(movie *db.Movie, tmdbID int) error {
		movie.TmdbID = tmdbID
		movie.Title = "North of the Sun"
		return nil
	}
	metadataCtx := app.NewTestingMDContext(&tmdbAgent)
	r := NewResolver(metadataCtx)

	movie := db.Movie{
		BaseItem: db.BaseItem{
			TmdbID: 1234,
		},
	}
	db.SaveMovie(&movie)

	subCh := r.MoviesChanged(context.Background())

	metadataCtx.MetadataManager.RefreshMovieMetadata(&movie)

	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Timeout waiting for MovieUpdatedEvent")
	case e := <-subCh:
		eventResolver, ok := e.ToMovieUpdatedEvent()
		assert.True(t, ok)
		assert.NotNil(t, eventResolver.Movie())
		assert.EqualValues(t, "North of the Sun", eventResolver.Movie().Title())
	}
}

func TestResolver_MoviesChanged_GarbageCollectMovie(t *testing.T) {
	metadataCtx := app.NewTestingMDContext(nil)
	r := NewResolver(metadataCtx)

	movie := db.Movie{
		Title: "Test Movie",
	}
	db.SaveMovie(&movie)

	subCh := r.MoviesChanged(context.Background())

	metadataCtx.MetadataManager.GarbageCollectMovieIfRequired(movie.ID)

	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Timeout waiting for MovieDeletedEvent")
	case e := <-subCh:
		eventResolver, ok := e.ToMovieDeletedEvent()
		assert.True(t, ok)
		assert.NotNil(t, eventResolver.MovieUUID())
	}
}

func TestResolver_SeasonDeleted_GarbageCollectEpisode(t *testing.T) {
	metadataCtx := app.NewTestingMDContext(nil)
	r := NewResolver(metadataCtx)

	series := &db.Series{
		Name: "Test Series",
	}
	db.SaveSeries(series)

	season := &db.Season{
		SeasonNumber: 1,
		SeriesID:     series.ID,
	}
	db.SaveSeason(season)

	episode := &db.Episode{
		Name:     "Test Episode",
		SeasonID: season.ID,
	}
	db.SaveEpisode(episode)

	seriesSubCh := r.SeriesChanged(context.Background())
	seasonSubCh, _ := r.SeasonChanged(context.Background(),
		&seasonChangedArgs{SeriesUUID: &series.UUID})

	// Episode does not have an EpisodeFile, so the whole tree should be removed
	metadataCtx.MetadataManager.GarbageCollectEpisodeIfRequired(episode.ID)

	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Timeout waiting for event")
	case e := <-seriesSubCh:
		eventResolver, ok := e.ToSeriesDeletedEvent()
		assert.True(t, ok)
		assert.EqualValues(t, series.UUID, eventResolver.SeriesUUID())
	}

	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Timeout waiting for event")
	case e := <-seasonSubCh:
		eventResolver, ok := e.ToSeasonDeletedEvent()
		assert.True(t, ok)
		assert.EqualValues(t, season.UUID, eventResolver.SeasonUUID())
	}
}
