package resolvers

import (
	"github.com/graph-gophers/graphql-go/relay"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers"
	"net/http"
)

type handler struct {
}

func (h handler) LetMeKnow() string {
	return "Yes"
}

// NewResolver is a new resolver, UPDATE THIS
func NewResolver(env *app.MetadataContext) *Resolver {
	r := &Resolver{exitChan: env.ExitChan}
	w := managers.NewDefaultWorkerPool()
	r.pool = w
	w.Handler = handler{}
	libs := db.AllLibraries()
	for i := range libs {
		r.AddLibraryManager(&libs[i])
	}
	return r
}

// AddLibraryManager adds a new manager
func (r *Resolver) AddLibraryManager(lib *db.Library) {
	man := managers.NewLibraryManager(lib, r.pool, r.exitChan)
	r.libs = append(r.libs, man)
	go man.RefreshAll()
}

// Resolver container object for all resolvers.
type Resolver struct {
	env      *app.MetadataContext
	pool     *managers.WorkerPool
	libs     []*managers.LibraryManager
	exitChan chan bool
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
func NewRelayHandler(env *app.MetadataContext) *relay.Handler {
	schema := InitSchema(env)

	return &relay.Handler{Schema: schema}
}
