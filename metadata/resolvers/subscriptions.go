package resolvers

import (
	"context"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"math/rand"
	"time"
)

type graphqlLibrarySubscriber struct {
	resolver *Resolver
}

func (h graphqlLibrarySubscriber) MovieAdded(movie *db.Movie) {
	e := &MovieAddedEvent{movie: &MovieResolver{*movie}}
	go func() {
		select {
		case h.resolver.movieAddedEvents <- e:
		case <-time.After(1 * time.Second):
		}
	}()
	return
}

func (h graphqlLibrarySubscriber) EpisodeAdded(episode *db.Episode) {
	e := &EpisodeAddedEvent{episode: &EpisodeResolver{Episode{Episode: *episode}}}
	go func() {
		select {
		case h.resolver.episodeAddedEvents <- e:
		case <-time.After(1 * time.Second):
		}
	}()
	return
}

// MovieAdded creates a subscription for all MovieAdded events
func (r *Resolver) MovieAdded(ctx context.Context) <-chan *MovieAddedEvent {
	log.Debugln("Adding subscription to MovieAddedEvent")
	c := make(chan *MovieAddedEvent)
	r.movieAddedSubscribers <- &movieAddedSubscriber{events: c, stop: ctx.Done()}

	return c
}

// EpisodeAdded creates a subscription for all MovieAdded events
func (r *Resolver) EpisodeAdded(ctx context.Context) <-chan *EpisodeAddedEvent {
	log.Debugln("Adding subscription to EpisodeAddedEvent")
	c := make(chan *EpisodeAddedEvent)
	r.epAddedSubChan <- &episodeAddedSubscriber{events: c, stop: ctx.Done()}

	return c
}

type episodeAddedSubscriber struct {
	stop   <-chan struct{}
	events chan<- *EpisodeAddedEvent
}

type movieAddedSubscriber struct {
	stop   <-chan struct{}
	events chan<- *MovieAddedEvent
}

// EpisodeAddedEvent blabla
type EpisodeAddedEvent struct {
	episode *EpisodeResolver
}

// Episode is a resolver for the episode struct
func (e *EpisodeAddedEvent) Episode() *EpisodeResolver {
	return e.episode
}

// MovieAddedEvent adds an event when a movie gets added
type MovieAddedEvent struct {
	movie *MovieResolver
}

// Movie is a resolver for the movie struct
func (m *MovieAddedEvent) Movie() *MovieResolver {
	return m.movie
}

func pubMovieAdded(id string, s *movieAddedSubscriber, e *MovieAddedEvent) {
	// The double select here: https://github.com/matiasanaya/go-graphql-subscription-example/issues/4#issuecomment-424604826
	select {
	case <-s.stop:
		//unsubscribe <- id
		return
	default:
	}

	select {
	case <-s.stop:
		//unsubscribe <- id
	case s.events <- e:
	case <-time.After(time.Second):
	}
}

func pubEpisodeAdded(id string, s *episodeAddedSubscriber, e *EpisodeAddedEvent) {
	// The double select here: https://github.com/matiasanaya/go-graphql-subscription-example/issues/4#issuecomment-424604826
	select {
	case <-s.stop:
		//unsubscribe <- id
		return
	default:
	}

	select {
	case <-s.stop:
		//unsubscribe <- id
	case s.events <- e:
	case <-time.After(time.Second):
	}
}

func (r *Resolver) broadcaster() {
	subscriptions := map[string]interface{}{}
	for {
		select {
		case s := <-r.movieAddedSubscribers:
			i := randomID()
			subscriptions[i] = s
			log.Println("Added movie subscription to broadcaster()")
		case s := <-r.epAddedSubChan:
			i := randomID()
			subscriptions[i] = s

		case e := <-r.episodeAddedEvents:
			log.Println("got episode event")
			for id, s := range subscriptions {
				episodeAddedSubscriber, ok := s.(*episodeAddedSubscriber)
				if ok {
					go pubEpisodeAdded(id, episodeAddedSubscriber, e)
				}
			}

		case e := <-r.movieAddedEvents:
			log.Println("got movie event")
			for id, s := range subscriptions {
				movieAddedSubscriber, ok := s.(*movieAddedSubscriber)
				if ok {
					go pubMovieAdded(id, movieAddedSubscriber, e)
				}
			}
		}
	}
}

func (r *Resolver) broadcastMovieAdded() {
	movieAddedSubs := map[string]*movieAddedSubscriber{}

	unsubscribe := make(chan string)

	for {
		select {
		case id := <-unsubscribe:
			delete(movieAddedSubs, id)
		case s := <-r.movieAddedSubscribers:
			movieAddedSubs[randomID()] = s
		case e := <-r.movieAddedEvents:
			for id, s := range movieAddedSubs {
				go func(id string, s *movieAddedSubscriber) {
					// The double select here: https://github.com/matiasanaya/go-graphql-subscription-example/issues/4#issuecomment-424604826
					select {
					case <-s.stop:
						unsubscribe <- id
						return
					default:
					}

					select {
					case <-s.stop:
						unsubscribe <- id
					case s.events <- e:
					case <-time.After(time.Second):
					}
				}(id, s)
			}
		}
	}
}

func randomID() string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, 16)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}
