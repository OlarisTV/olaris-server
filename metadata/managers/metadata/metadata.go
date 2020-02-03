package metadata

import (
	"runtime"
	"sync"

	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/db"
	mhelpers "gitlab.com/olaris/olaris-server/metadata/helpers"
)

// MetadataManager manages the metadata repository that is referenced by the files in the various
// libraries.
type MetadataManager struct {
	seriesCreationMutex sync.Mutex
	moviesCreationMutex sync.Mutex

	// Read/write lock for episode manipulation
	// TODO(Leon Handreke): Use proper locking,
	//  everywhere. This is a quickfix for the garbage collection routine.
	episodeLock sync.Map
	seasonLock  sync.Map
	seriesLock  sync.Map

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
	wg := &sync.WaitGroup{}
	uuids := make(chan string, runtime.NumCPU())

	missingItems := db.ItemsWithMissingMetadata()
	log.Debugln(len(missingItems), " items appear to be missing art.")
	for _, UUID := range missingItems {
		go func(UUID string) {
			wg.Add(1)
			defer wg.Done()
			uuids <- UUID
			m.RefreshAgentMetadataForUUID(UUID)
			<-uuids
		}(UUID)
	}

	wg.Wait()
	return
}

// RefreshAgentMetadataForUUID takes an UUID of a mediaitem and refreshes all metadata
func (m *MetadataManager) RefreshAgentMetadataForUUID(UUID string) bool {

	log.WithFields(log.Fields{"uuid": UUID}).
		Debugln("Looking to refresh metadata agent data.")
	movie, err := db.FindMovieByUUID(UUID)
	if err == nil {
		mhelpers.WithLock(func() {
			m.refreshAndSaveMovieMetadata(movie)
		}, movie.UUID)
		return true
	}

	series, err := db.FindSeriesByUUID(UUID)
	if err == nil {
		mhelpers.WithLock(func() {
			m.UpdateSeriesMD(series)
		}, series.UUID)
		return true
	}

	season, err := db.FindSeasonByUUID(UUID)
	if err == nil {
		mhelpers.WithLock(func() {
			m.UpdateSeasonMD(season)
		}, season.UUID)
		return true
	}

	episode, err := db.FindEpisodeByUUID(UUID)
	if err == nil {
		mhelpers.WithLock(func() {
			m.UpdateEpisodeMD(episode)
		}, episode.UUID)
		return true
	}

	return false
}
