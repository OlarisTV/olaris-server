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

var router = mux.NewRouter()

func GetHandler() http.Handler {
	router.PathPrefix("/player").Handler(http.StripPrefix("/player", http.FileServer(assetFS())))
	router.HandleFunc("/api/v1/files", serveFileIndex)
	router.HandleFunc("/api/v1/state", handleSetMediaPlaybackState).Methods("POST")
	// Currently, we serve these as two different manifests because switching doesn't work at all with misaligned
	// segments.
	router.HandleFunc("/{fileLocator:.*}/hls-transmuxing-manifest.m3u8", serveHlsTransmuxingMasterPlaylist)
	router.HandleFunc("/{fileLocator:.*}/hls-transcoding-manifest.m3u8", serveHlsTranscodingMasterPlaylist)
	router.HandleFunc("/{fileLocator:.*}/hls-manifest.m3u8", serveHlsMasterPlaylist)
	router.HandleFunc("/{fileLocator:.*}/{streamId}/{representationId}/media.m3u8", serveHlsTranscodingMediaPlaylist)
	router.HandleFunc("/{fileLocator:.*}/{streamId}/{representationId}/{segmentId:[0-9]+}.m4s", serveSegment)
	router.HandleFunc("/{fileLocator:.*}/{streamId}/{representationId}/init.mp4", serveInit)

	router.HandleFunc("/rclone/{rcloneRemote}/{rclonePath:.*}", serveRcloneFile).
		Name("rcloneFile")

	//TODO: (Maran) This is probably not serving subfolders yet
	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(*mediaFilesDir))))

	handler := cors.AllowAll().Handler(router)
	return handler
}

func Cleanup() {
	for _, s := range sessions {
		s.Destroy()
	}
}
