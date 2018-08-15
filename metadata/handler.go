package metadata

import (
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/resolvers"

	"gitlab.com/olaris/olaris-server/metadata/auth"
	"net/http"
)

func GetHandler(menv *db.MetadataContext) http.Handler {
	imageManager := NewImageManager()

	r := mux.NewRouter()
	r.Handle("/query", auth.AuthMiddleWare(resolvers.NewRelayHandler(menv)))

	r.Handle("/v1/auth", http.HandlerFunc(auth.UserHandler)).Methods("POST")

	r.Handle("/v1/user", http.HandlerFunc(auth.CreateUserHandler)).Methods("POST")
	r.Handle("/v1/user/setup", http.HandlerFunc(auth.ReadyForSetup))

	// TODO(Maran): This should be authenticated too.
	r.Handle("/images/{provider}/{size}/{id}", http.HandlerFunc(imageManager.HttpHandler))

	handler := cors.AllowAll().Handler(r)

	return handler
}
