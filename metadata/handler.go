package metadata

import (
	"github.com/gorilla/mux"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/resolvers"
	"net/http"
)

func GetHandler() http.Handler {
	mctx := db.NewMDContext()
	defer mctx.Db.Close()

	libraryManager := NewLibraryManager(mctx)
	libraryManager.ActivateAll()

	imageManager := NewImageManager(mctx)

	r := mux.NewRouter()

	r.Handle("/query", resolvers.NewRelayHandler(mctx))
	r.Handle("/images/{provider}/{size}/{id}", http.HandlerFunc(imageManager.HttpHandler))
	r.HandleFunc("/graphiql", http.HandlerFunc(resolvers.GraphiQLHandler))

	return r
}
