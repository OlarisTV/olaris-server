package main

//go:generate go-bindata-assetfs -pkg $GOPACKAGE static/...

import (
	"context"
	"flag"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/peak6/envflag"
	"github.com/rs/cors"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"gitlab.com/bytesized/bytesized-streaming/metadata"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"time"
)

var mediaFilesDir = flag.String("media_files_dir", "/var/media", "Path to the media files to be served")

var sessions = []*ffmpeg.TranscodingSession{}

// Read-modify-write mutex for sessions. This ensures that two parallel requests don't both create a session.
var sessionsMutex = sync.Mutex{}

func main() {
	flag.Parse()
	envflag.Parse()

	mctx := metadata.NewMDContext()
	defer mctx.Db.Close()
	libraryManager := metadata.NewLibraryManager(mctx)
	libraryManager.ActivateAll()

	imageManager := metadata.NewImageManager(mctx)
	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	r := mux.NewRouter()
	r.PathPrefix("/player").Handler(http.StripPrefix("/player", http.FileServer(assetFS())))
	r.HandleFunc("/api/v1/files", serveFileIndex)
	r.HandleFunc("/api/v1/state", handleSetMediaPlaybackState).Methods("POST")
	r.Handle("/query", metadata.NewRelayHandler(mctx))
	r.Handle("/images/{provider}/{size}/{id}", http.HandlerFunc(imageManager.HttpHandler))
	r.HandleFunc("/graphiql", http.HandlerFunc(metadata.GraphiQLHandler))
	// Currently, we serve these as two different manifests because switching doesn't work at all with misaligned
	// segments.
	r.HandleFunc("/{filename:.*}/hls-transmuxing-manifest.m3u8", serveHlsTransmuxingMasterPlaylist)
	r.HandleFunc("/{filename:.*}/hls-transcoding-manifest.m3u8", serveHlsTranscodingMasterPlaylist)
	r.HandleFunc("/{filename:.*}/hls-manifest.m3u8", serveHlsMasterPlaylist)
	r.HandleFunc("/{filename:.*}/{streamId}/{representationId}/media.m3u8", serveHlsTranscodingMediaPlaylist)
	r.HandleFunc("/{filename:.*}/{streamId}/{representationId}/{segmentId:[0-9]+}.m4s", serveSegment)
	r.HandleFunc("/{filename:.*}/{streamId}/{representationId}/init.mp4", serveInit)

	//TODO: (Maran) This is probably not serving subfolders yet
	r.Handle("/", http.FileServer(http.Dir(*mediaFilesDir)))

	var handler http.Handler
	handler = r
	handler = cors.AllowAll().Handler(handler)
	handler = handlers.LoggingHandler(os.Stdout, handler)

	var port = os.Getenv("PORT")
	// Set a default port if there is nothing in the environment
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{Addr: ":" + port, Handler: handler}
	go srv.ListenAndServe()

	// Wait for termination signal
	<-stopChan

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)

	for _, s := range sessions {
		s.Destroy()

	}
}
