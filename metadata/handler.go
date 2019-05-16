// Package metadata implements metadata server features such as media indexing,
// media metadata lookup on external services and exposing this data via APIs.
package metadata

import (
	"github.com/gorilla/mux"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/resolvers"

	"gitlab.com/olaris/olaris-server/metadata/auth"
)

// RegisterRoutes defines the handlers for metadata endpoints such as graphql and REST methods.
func RegisterRoutes(menv *app.MetadataContext, r *mux.Router) {
	imageManager := NewImageManager()

	r.Handle("/query", auth.MiddleWare(resolvers.NewRelayHandler(menv)))

	r.HandleFunc("/v1/auth", auth.UserHandler).Methods("POST")

	r.HandleFunc("/v1/user", auth.CreateUserHandler).Methods("POST")
	r.HandleFunc("/v1/user/setup", auth.ReadyForSetup)

	// TODO(Maran): This should be authenticated too.
	r.HandleFunc("/images/{provider}/{size}/{id}", imageManager.HTTPHandler)

	// TODO(Maran): Check if Cors is still working
	//	handler := cors.AllowAll().Handler(r)
}
