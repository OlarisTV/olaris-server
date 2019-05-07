package streaming

import (
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"net/http"
)

var router = mux.NewRouter()

func GetHandler() http.Handler {
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/hls-transmuxing-manifest.m3u8", serveHlsTransmuxingMasterPlaylist)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/hls-transcoding-manifest.m3u8", serveHlsTranscodingMasterPlaylist)
	router.HandleFunc("/files/{fileLocator:.*}/metadata.json", serveMetadata)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/hls-manifest.m3u8", serveHlsMasterPlaylist)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/dash-manifest.mpd", serveDASHManifest)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/media.m3u8", serveHlsTranscodingMediaPlaylist)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/{segmentId:[0-9]+}.m4s", serveMediaSegment)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/{segmentId:[0-9]+}.vtt", serveSubtitleSegment)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/init.mp4", serveInit)
	router.HandleFunc("/ffmpeg/{playbackSessionID}/feedback", serveFFmpegFeedback)

	// This handler just serves up the file for downloading. This is also used
	// internally by ffmpeg to access rclone files.
	router.HandleFunc("/files/{fileLocator:.*}", serveFile)

	router.HandleFunc("/debug/playbackSessions", servePlaybackSessionDebugPage)

	handler := cors.AllowAll().Handler(router)
	return handler
}

func Cleanup() {
	for _, s := range playbackSessions {
		s.referenceCount--
		s.CleanupIfRequired()

		if s.referenceCount > 0 {
			log.Warn("Playback session reference count leak: ", s.TranscodingSession)
		}
	}
}
