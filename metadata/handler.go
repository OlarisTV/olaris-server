// Package metadata implements metadata server features such as media indexing,
// media metadata lookup on external services and exposing this data via APIs.
package metadata

import (
	"github.com/gorilla/mux"
	"github.com/graph-gophers/graphql-transport-ws/graphqlws"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/resolvers"
	"net/http"

	"gitlab.com/olaris/olaris-server/metadata/auth"
)

// RegisterRoutes defines the handlers for metadata endpoints such as graphql and REST methods.
func RegisterRoutes(menv *app.MetadataContext, r *mux.Router) {
	imageManager := NewImageManager()

	schema, handler := resolvers.NewRelayHandler(menv)
	r.Handle("/query", auth.MiddleWare(graphqlws.NewHandlerFunc(schema, handler)))

	r.HandleFunc("/v1/auth", auth.UserHandler).Methods("POST")

	r.HandleFunc("/v1/version", versionHandler).Methods("GET")

	r.HandleFunc("/v1/user", auth.CreateUserHandler).Methods("POST")
	r.HandleFunc("/v1/user/setup", auth.ReadyForSetup)

	// TODO(Maran): This should be authenticated too.
	r.HandleFunc("/images/{provider}/{size}/{id}", imageManager.HTTPHandler)
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(helpers.Version))
}
