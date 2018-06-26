package metadata

import (
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/resolvers"

	"net/http"
	"gitlab.com/bytesized/bytesized-streaming/metadata/auth"
)

func GetHandler(menv *db.MetadataContext) http.Handler {
	imageManager := NewImageManager()

	r := mux.NewRouter()
	r.Handle("/query", auth.AuthMiddleWare(resolvers.NewRelayHandler(menv)))

	r.Handle("/v1/auth", http.HandlerFunc(auth.UserHandler)).Methods("POST")

	r.Handle("/v1/user", http.HandlerFunc(auth.CreateUserHandler)).Methods("POST")

	// TODO(Maran): This should be authenticated too.
	r.Handle("/images/{provider}/{size}/{id}", http.HandlerFunc(imageManager.HttpHandler))

	handler := cors.AllowAll().Handler(r)

	return handler
}
