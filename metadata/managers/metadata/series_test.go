package metadata

import (
	"github.com/ryanbradynd05/go-tmdb"
	"github.com/stretchr/testify/assert"
	"gitlab.com/olaris/olaris-server/metadata/agents/agentsfakes"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"testing"
)

// TODO(Leon Handreke): Merge this into our TMDB lib
type tvSearchResult struct {
	BackdropPath  string `json:"backdrop_path"`
	ID            int
	OriginalName  string   `json:"original_name"`
	FirstAirDate  string   `json:"first_air_date"`
	OriginCountry []string `json:"origin_country"`
	PosterPath    string   `json:"poster_path"`
	Popularity    float32
	Name          string
	VoteAverage   float32 `json:"vote_average"`
	VoteCount     uint32  `json:"vote_count"`
}

func TestMetadataManager_GetOrCreateEpisodeForEpisodeFile(t *testing.T) {
	// TODO(Leon Handreke): Dependency inject instead of relying on global singletons
	db.NewInMemoryDBForTests(false)
	agent := agentsfakes.FakeMetadataRetrievalAgent{}
	m := MetadataManager{
		agent: &agent,
	}

	episodeFile := db.EpisodeFile{
		MediaItem: db.MediaItem{
			FileName: "The Walking Dead S01E01.mkv",
			FilePath: "local#/The Walking Dead S01E01.mkv",
		},
	}
	// This is what TMDB really does and why we have the string distance search feature
	agent.TmdbSearchTvStub = func(name string, options map[string]string) (
		*tmdb.TvSearchResults, error) {
		return &tmdb.TvSearchResults{
			Results: []struct {
				BackdropPath  string `json:"backdrop_path"`
				ID            int
				OriginalName  string   `json:"original_name"`
				FirstAirDate  string   `json:"first_air_date"`
				OriginCountry []string `json:"origin_country"`
				PosterPath    string   `json:"poster_path"`
				Popularity    float32
				Name          string
				VoteAverage   float32 `json:"vote_average"`
				VoteCount     uint32  `json:"vote_count"`
			}{
				{Name: "Fear the Walking Dead", ID: 1},
				{Name: "The Walking Dead", ID: 2},
			},
		}, nil
	}
	agent.UpdateEpisodeMDStub = func(
		episode *db.Episode, seriesTMDBID int, seasonNum int, episodeNum int) error {
		if seriesTMDBID == 1 {
			episode.TmdbID = 101
		} else if seriesTMDBID == 2 {
			episode.TmdbID = 102
		}
		return nil
	}

	episode, err := m.GetOrCreateEpisodeForEpisodeFile(&episodeFile)
	assert.Nil(t, err)
	assert.Equal(t, 102, episode.TmdbID)
}
