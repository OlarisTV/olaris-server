package metadata

import (
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/db"
	mhelpers "gitlab.com/olaris/olaris-server/metadata/helpers"
	"sync"
)

// MetadataManager manages the metadata repository that is referenced by the files in the various
// libraries.
type MetadataManager struct {
	seriesMutex sync.Mutex
	moviesMutex sync.Mutex

	Subscriber LibrarySubscriber
	agent      agents.MetadataRetrievalAgent
}

// NewMetadataManager creates a new MetadataManager
func NewMetadataManager(agent agents.MetadataRetrievalAgent) *MetadataManager {
	return &MetadataManager{
		agent: agent,
	}
}

// RefreshAgentMetadataWithMissingArt loops over all series/episodes/seasons and movies with missing art (posters/backdrop) and tries to retrieve them.
func (m *MetadataManager) RefreshAgentMetadataWithMissingArt() {
	log.Debugln("Checking and updating media items for missing art.")
	for _, UUID := range db.ItemsWithMissingMetadata() {
		m.RefreshAgentMetadataForUUID(UUID)
	}
}

// RefreshAgentMetadataForUUID takes an UUID of a mediaitem and refreshes all metadata
func (m *MetadataManager) RefreshAgentMetadataForUUID(UUID string) bool {

	log.WithFields(log.Fields{"uuid": UUID}).
		Debugln("Looking to refresh metadata agent data.")
	movie, err := db.FindMovieByUUID(UUID)
	if err != nil {
		go mhelpers.WithLock(func() {
			m.UpdateMovieMD(movie)
		}, movie.UUID)
		return true
	}

	series, err := db.FindSeriesByUUID(UUID)
	if err != nil {
		go mhelpers.WithLock(func() {
			m.UpdateSeriesMD(series)
		}, series.UUID)
		return true
	}

	season, err := db.FindSeasonByUUID(UUID)
	if err != nil {
		go mhelpers.WithLock(func() {
			m.UpdateSeasonMD(season)
		}, season.UUID)
		return true
	}

	episode, err := db.FindEpisodeByUUID(UUID)
	if err != nil {
		go mhelpers.WithLock(func() {
			m.UpdateEpisodeMD(episode)
		}, episode.UUID)
		return true
	}
	return false
}
