package metadata

import "sync"

type MetadataEventType int

const (
	MetadataEventTypeMovieAdded   = iota // payload *db.Movie
	MetadataEventTypeMovieUpdated        // payload *db.Movie
	MetadataEventTypeMovieDeleted        // payload: *db.Movie

	MetadataEventTypeEpisodeAdded   // payload *db.Episode
	MetadataEventTypeEpisodeUpdated // payload *db.Episode
	MetadataEventTypeEpisodeDeleted // payload *db.Episode

	MetadataEventTypeSeasonAdded   // payload *db.Season
	MetadataEventTypeSeasonUpdated // payload *db.Season
	MetadataEventTypeSeasonDeleted // payload *db.Season

	MetadataEventTypeSeriesAdded   // payload *db.Series
	MetadataEventTypeSeriesUpdated // payload *db.Series
	MetadataEventTypeSeriesDeleted // payload *db.Series
)

type MetadataEvent struct {
	EventType MetadataEventType
	Payload   interface{}
}

type MetadataSubscriber chan *MetadataEvent

type metadataEventBroker struct {
	subscribers      map[MetadataSubscriber]struct{}
	subscribersMutex sync.RWMutex
}

func newMetadataEventBroker() *metadataEventBroker {
	return &metadataEventBroker{
		subscribers: map[MetadataSubscriber]struct{}{},
	}
}

func (broker *metadataEventBroker) addSubscriber() MetadataSubscriber {
	broker.subscribersMutex.Lock()
	defer broker.subscribersMutex.Unlock()

	ch := make(MetadataSubscriber, 2)
	broker.subscribers[ch] = struct{}{}

	return ch
}

func (broker *metadataEventBroker) removeSubscriber(s MetadataSubscriber) {
	broker.subscribersMutex.Lock()
	defer broker.subscribersMutex.Unlock()

	ch := make(MetadataSubscriber, 2)
	broker.subscribers[ch] = struct{}{}

	close(s)
}

func (broker *metadataEventBroker) publish(e *MetadataEvent) {
	broker.subscribersMutex.RLock()
	defer broker.subscribersMutex.RUnlock()

	for s := range broker.subscribers {
		s <- e
	}
}

// AddSubscriber adds an event subscriber to this MetadataManager
func (m *MetadataManager) AddSubscriber() MetadataSubscriber {
	return m.eventBroker.addSubscriber()
}

// RemoveSubscriber removes an event subscriber from this MetadataManager
func (m *MetadataManager) RemoveSubscriber(s MetadataSubscriber) {
	m.eventBroker.removeSubscriber(s)
}
