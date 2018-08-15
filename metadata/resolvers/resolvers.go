package resolvers

import (
	"github.com/graph-gophers/graphql-go/relay"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"net/http"
)

type Resolver struct {
	env *db.MetadataContext
}

type ErrorResolver struct {
	r Error
}

type Error struct {
	message  string
	hasError bool
}

func CreateErr(err error) Error {
	return Error{message: err.Error(), hasError: true}
}

func CreateErrResolver(err error) *ErrorResolver {
	return &ErrorResolver{r: Error{message: err.Error(), hasError: true}}
}

func (r *ErrorResolver) Message() string {
	return r.r.message
}
func (r *ErrorResolver) HasError() bool {
	return r.r.hasError
}

func GraphiQLHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(graphiQLpage)
}

func NewRelayHandler(env *db.MetadataContext) *relay.Handler {
	schema := InitSchema(env)
	return &relay.Handler{Schema: schema}
}
