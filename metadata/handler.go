package metadata

import (
	"github.com/gorilla/mux"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/resolvers"
	"net/http"
)

func GetHandler(mctx *db.MetadataContext) http.Handler {
	imageManager := NewImageManager()

	r := mux.NewRouter()
	r.Handle("/query", db.AuthMiddleWare(resolvers.NewRelayHandler(mctx)))

	r.Handle("/v1/auth", http.HandlerFunc(db.AuthHandler))

	r.Handle("/v1/user", http.HandlerFunc(db.CreateUserHandler))

	r.Handle("/images/{provider}/{size}/{id}", http.HandlerFunc(imageManager.HttpHandler))

	return r
}
