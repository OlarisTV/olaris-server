package metadata

import (
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/resolvers"

	"net/http"
)

func GetHandler(mctx *db.MetadataContext) http.Handler {
	imageManager := NewImageManager()

	r := mux.NewRouter()
	r.Handle("/query", db.AuthMiddleWare(resolvers.NewRelayHandler(mctx)))

	r.Handle("/v1/auth", http.HandlerFunc(db.AuthHandler)).Methods("POST")

	r.Handle("/v1/user", http.HandlerFunc(db.CreateUserHandler)).Methods("POST")

	r.Handle("/images/{provider}/{size}/{id}", http.HandlerFunc(imageManager.HttpHandler))

	handler := cors.AllowAll().Handler(r)

	return handler
}
