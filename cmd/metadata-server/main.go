package main

import (
	"flag"
	"github.com/gorilla/mux"
	"gitlab.com/bytesized/bytesized-streaming/metadata"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"net/http"
)

var mediaFilesDir = flag.String("media_files_dir", "/var/media", "Path used if no libraries exist for the default library")

func main() {
	flag.Parse()

	mctx := db.NewMDContext()
	defer mctx.Db.Close()

	libraryManager := db.NewLibraryManager()
	// Scan on start-up
	go libraryManager.RefreshAll()

	r := mux.NewRouter()
	r.PathPrefix("/m").Handler(http.StripPrefix("/m", metadata.GetHandler(mctx)))

	srv := &http.Server{Addr: ":8080", Handler: r}
	srv.ListenAndServe()
}
