package streaming

//go:generate go-bindata-assetfs -pkg $GOPACKAGE static/...

import (
	"flag"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"net/http"
	"sync"
)

var mediaFilesDir = flag.String("media_files_dir", "/var/media", "Path to the media files to be served")

var sessions = []*ffmpeg.TranscodingSession{}

// Read-modify-write mutex for sessions. This ensures that two parallel requests don't both create a session.
var sessionsMutex = sync.Mutex{}

func GetHandler() http.Handler {
	r := mux.NewRouter()

	r.PathPrefix("/player").Handler(http.StripPrefix("/player", http.FileServer(assetFS())))
	r.HandleFunc("/api/v1/files", serveFileIndex)
	r.HandleFunc("/api/v1/state", handleSetMediaPlaybackState).Methods("POST")
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

	handler := cors.AllowAll().Handler(r)
	return handler
}

func Cleanup() {
	for _, s := range sessions {
		s.Destroy()
	}
}
