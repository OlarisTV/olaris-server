package streaming

import (
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"net/http"
	"sync"
)

// Read-modify-write mutex for sessions. This ensures that two parallel requests don't both create a session.
var sessionsMutex = sync.Mutex{}

var router = mux.NewRouter()

func GetHandler() http.Handler {
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/hls-transmuxing-manifest.m3u8", serveHlsTransmuxingMasterPlaylist)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/hls-transcoding-manifest.m3u8", serveHlsTranscodingMasterPlaylist)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/hls-manifest.m3u8", serveHlsMasterPlaylist)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/dash-manifest.mpd", serveDASHManifest)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/media.m3u8", serveHlsTranscodingMediaPlaylist)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/{segmentId:[0-9]+}.m4s", serveMediaSegment)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/{segmentId:[0-9]+}.vtt", serveSubtitleSegment)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/init.mp4", serveInit)

	// This handler just serves up the file for downloading. This is also used
	// internally by ffmpeg to access rclone files.
	router.HandleFunc("/files/{fileLocator:.*}", serveFile)

	handler := cors.AllowAll().Handler(router)
	return handler
}

func Cleanup() {
	for _, s := range playbackSessions {
		s.Cleanup()
	}
}
