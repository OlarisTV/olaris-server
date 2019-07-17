package resolvers

import (
	"context"
	"github.com/stretchr/testify/assert"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

func TestPlayState(t *testing.T) {
	const testUserID = 1

	ctx := auth.ContextWithUserID(context.Background(), testUserID)
	r := NewResolver(app.NewMDContext(db.InMemory, true, true))

	mi := db.MediaItem{FilePath: "/tmp/test", Title: "Test.mkv"}
	stream := db.Stream{CodecName: "test"}
	mf := db.MovieFile{MediaItem: mi, Streams: []db.Stream{stream}}

	movie := db.Movie{
		BaseItem: db.BaseItem{
			TmdbID: 123,
		},
		Title:         "Test Title",
		OriginalTitle: "Mad Max: Road Fury",
		MovieFiles:    []db.MovieFile{mf},
	}
	db.CreateMovie(&movie)
	db.SavePlayState(&db.PlayState{
		Finished: false, Playtime: 33,
		MediaUUID: movie.UUID, UserID: testUserID})
	// UUID gets created on insertion
	movieUUID := movie.UUID

	movieResolvers := r.Movies(ctx, &queryArgs{UUID: &movieUUID})
	assert.Len(t, movieResolvers, 1, "Should return exactly one movie")
	movieResolver := movieResolvers[0]

	assert.Equal(t, "Test Title", movieResolver.Title())

	assert.EqualValues(t, 33, movieResolver.PlayState(ctx).Playtime())
}
