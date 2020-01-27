package metadata

import (
	"github.com/ryanbradynd05/go-tmdb"
	"github.com/stretchr/testify/assert"
	"gitlab.com/olaris/olaris-server/metadata/agents/agentsfakes"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

func TestGetOrCreateMovieForMovieFile_SearchByStringDistance(t *testing.T) {
	// TODO(Leon Handreke): Dependency inject instead of relying on global singletons
	db.NewInMemoryDBForTests(false)
	agent := agentsfakes.FakeMetadataRetrievalAgent{}
	m := NewMetadataManager(&agent)

	movieFile := db.MovieFile{
		MediaItem: db.MediaItem{
			FileName: "The Walking Dead.mkv",
			FilePath: "local#/The Walking Dead.mkv",
		},
	}
	// This is what TMDB really does and why we have the string distance search feature
	agent.TmdbSearchMovieStub = func(name string, options map[string]string) (
		*tmdb.MovieSearchResults, error) {
		return &tmdb.MovieSearchResults{
			Results: []tmdb.MovieShort{
				{Title: "Fear the Walking Dead", ID: 1},
				{Title: "The Walking Dead", ID: 2},
			},
		}, nil
	}
	agent.UpdateMovieMDStub = func(movie *db.Movie, tmdbID int) error {
		if tmdbID == 1 {
			movie.Title = "Fear the Walking Dead"
		} else if tmdbID == 2 {
			movie.Title = "The Walking Dead"
		}
		return nil
	}

	movie, err := m.GetOrCreateMovieForMovieFile(&movieFile)
	assert.Nil(t, err)
	assert.Equal(t, "The Walking Dead", movie.Title)
}
