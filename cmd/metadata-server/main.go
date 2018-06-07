package main

import (
	"flag"
	"github.com/gorilla/mux"
	"gitlab.com/bytesized/bytesized-streaming/metadata"
	"net/http"
)

var mediaFilesDir = flag.String("media_files_dir", "/var/media", "Path used if no libraries exist for the default library")

func main() {
	flag.Parse()

	r := mux.NewRouter()
	r.PathPrefix("/m").Handler(http.StripPrefix("/m", metadata.GetHandler()))

	srv := &http.Server{Addr: ":8080", Handler: r}
	srv.ListenAndServe()

}
