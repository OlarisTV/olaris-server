package resolvers

import (
	"github.com/stretchr/testify/assert"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

func TestUnidentifiedEpisodeFiles(t *testing.T) {
	metadataCtx := app.NewTestingMDContext(nil)
	r := NewResolver(metadataCtx)

	db.CreateEpisode(&db.Episode{
		Name: "Test Episode",
		EpisodeFiles: []db.EpisodeFile{
			{
				MediaItem: db.MediaItem{
					FilePath: "local#/tmp/test1.mkv",
				},
			},
		},
	})
	metadataCtx.Db.Create(&db.EpisodeFile{
		MediaItem: db.MediaItem{
			FilePath: "local#/tmp/test2.mkv",
		},
	})

	response := r.UnidentifiedEpisodeFiles(&unidentifiedEpisodeFilesArgs{})

	assert.Len(t, response, 1)
	filePath, _ := response[0].FilePath()
	assert.Equal(t, "/tmp/test2.mkv", filePath)

	assert.EqualValues(t, db.BackendLocal, response[0].Library().Backend())
}
