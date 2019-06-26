package resolvers

import (
	"context"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers"
	"math/rand"
	"net/http"
	"time"
)

type graphQLNotificationHandler struct {
	resolver *Resolver
}

func (h graphQLNotificationHandler) MovieAdded(movie *db.Movie) {
	e := &movieAddedEvent{movie: &MovieResolver{*movie}}
	go func() {
		select {
		case h.resolver.movieAddedEvents <- e:
		case <-time.After(1 * time.Second):
		}
	}()
	return
}

// MovieAdded subscription
func (r *Resolver) MovieAdded(ctx context.Context) <-chan *movieAddedEvent {
	log.Debugln("Adding subscription to movieAddedEvent")
	c := make(chan *movieAddedEvent)
	r.movieAddedSubscriber <- &movieAddedSubscriber{events: c, stop: ctx.Done()}

	return c
}

// NewResolver is a new resolver, UPDATE THIS
func NewResolver(env *app.MetadataContext) *Resolver {
	r := &Resolver{exitChan: env.ExitChan, movieAddedSubscriber: make(chan *movieAddedSubscriber), movieAddedEvents: make(chan *movieAddedEvent)}

	w := managers.NewDefaultWorkerPool()
	r.pool = w

	w.Handler = graphQLNotificationHandler{resolver: r}

	libs := db.AllLibraries()
	for i := range libs {
		r.AddLibraryManager(&libs[i])
	}
	go r.broadcastMovieAdded()
	return r
}

// AddLibraryManager adds a new manager
func (r *Resolver) AddLibraryManager(lib *db.Library) {
	man := managers.NewLibraryManager(lib, r.pool, r.exitChan)
	r.libs = append(r.libs, man)
	go man.RefreshAll()
}

type movieAddedSubscriber struct {
	stop   <-chan struct{}
	events chan<- *movieAddedEvent
}

type movieAddedEvent struct {
	movie *MovieResolver
}

func (m *movieAddedEvent) Movie() *MovieResolver {
	return m.movie
}

// Resolver container object for all resolvers.
type Resolver struct {
	env                  *app.MetadataContext
	pool                 *managers.WorkerPool
	libs                 []*managers.LibraryManager
	exitChan             chan bool
	movieAddedEvents     chan *movieAddedEvent
	movieAddedSubscriber chan *movieAddedSubscriber
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

	// NOTE: subscribing and sending events are at odds.
	for {
		select {
		case id := <-unsubscribe:
			delete(subscribers, id)
		case s := <-r.movieAddedSubscriber:
			subscribers[randomID()] = s
		case e := <-r.movieAddedEvents:
			for id, s := range subscribers {
				go func(id string, s *movieAddedSubscriber) {
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

// ErrorResolver holds error information.
type ErrorResolver struct {
	r Error
}

// Error is a generic user level error.
type Error struct {
	message  string
	hasError bool
}

// CreateErr creates an error object with the given error.
func CreateErr(err error) Error {
	return Error{message: err.Error(), hasError: true}
}

// CreateErrResolver creates a resolver around the given error.
func CreateErrResolver(err error) *ErrorResolver {
	return &ErrorResolver{r: CreateErr(err)}
}

// Message returns the error message.
func (r *ErrorResolver) Message() string {
	return r.r.message
}

// HasError returns bool if an error has been found.
func (r *ErrorResolver) HasError() bool {
	return r.r.hasError
}

// GraphiQLHandler returns graphql handler.
func GraphiQLHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(graphiQLpage)
}

// NewRelayHandler handles graphql requests.
func NewRelayHandler(env *app.MetadataContext) (*graphql.Schema, *relay.Handler) {
	schema := InitSchema(env)
	handler := &relay.Handler{Schema: schema}
	return schema, handler
}
