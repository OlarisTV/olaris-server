// Package metadata implements metadata server features such as media indexing,
// media metadata lookup on external services and exposing this data via APIs.
package metadata

import (
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/resolvers"

	"gitlab.com/olaris/olaris-server/metadata/auth"
	"net/http"
)

// GetHandler defines the handlers for metadata endpoints such as graphql and REST methods.
func GetHandler(menv *app.MetadataContext) http.Handler {
	imageManager := NewImageManager()

	r := mux.NewRouter()
	r.Handle("/query", auth.MiddleWare(resolvers.NewRelayHandler(menv)))

	r.Handle("/v1/auth", http.HandlerFunc(auth.UserHandler)).Methods("POST")

	r.Handle("/v1/user", http.HandlerFunc(auth.CreateUserHandler)).Methods("POST")
	r.Handle("/v1/user/setup", http.HandlerFunc(auth.ReadyForSetup))

	// TODO(Maran): This should be authenticated too.
	r.Handle("/images/{provider}/{size}/{id}", http.HandlerFunc(imageManager.HTTPHandler))

	handler := cors.AllowAll().Handler(r)

	return handler
}
