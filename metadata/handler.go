package metadata

import (
	"github.com/gorilla/mux"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/resolvers"
	"net/http"
)

func GetHandler() http.Handler {
	mctx := db.NewMDContext()
	refresh := make(chan int)
	mctx.RefreshChan = refresh

	libraryManager := NewLibraryManager(mctx)
	libraryManager.ActivateAll()

	imageManager := NewImageManager(mctx)

	r := mux.NewRouter()

	r.Handle("/query", db.AuthMiddleWare(resolvers.NewRelayHandler(mctx)))
	r.Handle("/auth", http.HandlerFunc(db.AuthHandler))
	r.Handle("/images/{provider}/{size}/{id}", http.HandlerFunc(imageManager.HttpHandler))

	go func() {
		for _ = range refresh {
			libraryManager.ActivateAll()
		}
	}()

	return r
}
