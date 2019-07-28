package resolvers

import (
	"github.com/stretchr/testify/assert"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

func TestUnidentifiedEpisodeFiles(t *testing.T) {
	metadataCtx := app.NewMDContext(db.InMemory, false, false)
	r := NewResolver(metadataCtx)

	db.CreateEpisode(&db.Episode{
		Name: "Test Episode",
		EpisodeFiles: []db.EpisodeFile{
			{
				MediaItem: db.MediaItem{
					FilePath: "/tmp/test1.mkv",
				},
			},
		},
	})
	metadataCtx.Db.Create(&db.EpisodeFile{
		MediaItem: db.MediaItem{
			FilePath: "/tmp/test2.mkv",
		},
	})

	response := r.UnidentifiedEpisodeFiles(&unidentifiedEpisodeFilesArgs{})

	assert.Len(t, response, 1)
	assert.Equal(t, "/tmp/test2.mkv", response[0].FilePath())
}
