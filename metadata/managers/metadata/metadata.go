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

	agent agents.MetadataRetrievalAgent

	eventBroker *metadataEventBroker
}

// NewMetadataManager creates a new MetadataManager
func NewMetadataManager(agent agents.MetadataRetrievalAgent) *MetadataManager {
	return &MetadataManager{
		agent:       agent,
		eventBroker: newMetadataEventBroker(),
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
		wg.Add(1)
		go func(UUID string) {
			defer wg.Done()
			uuids <- UUID
			m.RefreshAgentMetadataForUUID(UUID)
			<-uuids
		}(UUID)
	}

	wg.Wait()
}

// RefreshAgentMetadataForUUID takes an UUID of a mediaitem and refreshes all metadata
func (m *MetadataManager) RefreshAgentMetadataForUUID(UUID string) bool {

	log.WithFields(log.Fields{"uuid": UUID}).
		Debugln("Looking to refresh metadata agent data.")
	movie, err := db.FindMovieByUUID(UUID)
	if err == nil {
		mhelpers.WithLock(func() {
			m.RefreshMovieMetadata(movie)
			db.SaveMovie(movie)
		}, movie.UUID)
		return true
	}

	series, err := db.FindSeriesByUUID(UUID)
	if err == nil {
		mhelpers.WithLock(func() {
			m.refreshSeriesMetadataFromAgent(series)
			db.SaveSeries(series)
		}, series.UUID)
		return true
	}

	season, err := db.FindSeasonByUUID(UUID)
	if err == nil {
		mhelpers.WithLock(func() {
			m.refreshSeasonMetadataFromAgent(season, season.GetSeries().TmdbID)
			db.SaveSeason(season)
		}, season.UUID)
		return true
	}

	episode, err := db.FindEpisodeByUUID(UUID)
	if err == nil {
		mhelpers.WithLock(func() {
			m.refreshEpisodeMetadataFromAgent(episode, episode.SeasonNum, episode.GetSeries().TmdbID)
			db.SaveEpisode(episode)
		}, episode.UUID)
		return true
	}

	return false
}
