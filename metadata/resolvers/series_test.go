package resolvers

import (
	"context"
	"github.com/stretchr/testify/assert"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

func TestEpisodePlayState(t *testing.T) {
	const testUserID = 1

	ctx := auth.ContextWithUserID(context.Background(), testUserID)
	r := NewResolver(app.NewMDContext(db.InMemory, true, true))

	mi := db.MediaItem{FilePath: "/tmp/test", Title: "Test.mkv"}
	stream := db.Stream{CodecName: "test"}
	episodeFile := db.EpisodeFile{MediaItem: mi, Streams: []db.Stream{stream}}

	episode := db.Episode{
		BaseItem: db.BaseItem{
			TmdbID: 123,
		},
		EpisodeFiles: []db.EpisodeFile{episodeFile},
		Name:         "Test Name",
	}
	db.CreateEpisode(&episode)
	db.SavePlayState(&db.PlayState{
		Finished: false, Playtime: 33,
		MediaUUID: episode.UUID, UserID: testUserID})
	// UUID gets created on insertion
	episodeUUID := episode.UUID

	episodeResolver := r.Episode(ctx, &mustUUIDArgs{UUID: &episodeUUID})

	assert.Equal(t, "Test Name", episodeResolver.Name())

	assert.EqualValues(t, 33, episodeResolver.PlayState(ctx).Playtime())
}
