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

// MovieAdded subscription
func (r *Resolver) MovieAdded(ctx context.Context) <-chan *MovieAddedEvent {
	log.Debugln("Adding subscription to MovieAddedEvent")
	c := make(chan *MovieAddedEvent)
	r.movieAddedSubscriber <- &movieAddedSubscriber{events: c, stop: ctx.Done()}

	return c
}

type movieAddedSubscriber struct {
	stop   <-chan struct{}
	events chan<- *MovieAddedEvent
}

type episodeAddedEvent struct {
	episode *EpisodeResolver
}

// MovieAddedEvent adds an event when a movie gets added
type MovieAddedEvent struct {
	movie *MovieResolver
}

// Movie is a resolver for the movie struct
func (m *MovieAddedEvent) Movie() *MovieResolver {
	return m.movie
}

func randomID() string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, 16)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func (r *Resolver) broadcastMovieAdded() {
	subscribers := map[string]*movieAddedSubscriber{}
	unsubscribe := make(chan string)

	for {
		select {
		case id := <-unsubscribe:
			delete(subscribers, id)
		case s := <-r.movieAddedSubscriber:
			subscribers[randomID()] = s
		case e := <-r.movieAddedEvents:
			for id, s := range subscribers {
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
