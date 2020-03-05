package streaming

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// For Apple devices to handle HLS properly, the m3u8 playlists must be sent with the correct Content-Type
func AddM3U8Header(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-mpegURL")
		next.ServeHTTP(w, r)
	})
}

// RegisterRoutes registers streaming routes to an existing router
func RegisterRoutes(router *mux.Router) {
	router.Handle("/files/{fileLocator:.*}/{sessionID}/hls-transmuxing-manifest.m3u8", AddM3U8Header(http.HandlerFunc(serveHlsTransmuxingMasterPlaylist)))
	router.Handle("/files/{fileLocator:.*}/{sessionID}/hls-transcoding-manifest.m3u8", AddM3U8Header(http.HandlerFunc(serveHlsTranscodingMasterPlaylist)))
	router.HandleFunc("/files/{fileLocator:.*}/metadata.json", serveMetadata)
	router.Handle("/files/{fileLocator:.*}/{sessionID}/hls-manifest.m3u8", AddM3U8Header(http.HandlerFunc(serveHlsMasterPlaylist)))
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/dash-manifest.mpd", serveDASHManifest)
	router.Handle("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/media.m3u8", AddM3U8Header(http.HandlerFunc(serveHlsTranscodingMediaPlaylist)))
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/{segmentId:[0-9]+}.m4s", serveMediaSegment)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/{segmentId:[0-9]+}.vtt", serveSubtitleSegment)
	router.HandleFunc("/files/{fileLocator:.*}/{sessionID}/{streamId}/{representationId}/init.mp4", serveInit)
	router.HandleFunc("/ffmpeg/{playbackSessionID}/feedback", serveFFmpegFeedback)

	// This handler just serves up the file for downloading. This is also used
	// internally by ffmpeg to access rclone files.
	router.HandleFunc("/files/{fileLocator:.*}", serveFile)

	router.HandleFunc("/debug/playbackSessions", servePlaybackSessionDebugPage)

	//handler := cors.AllowAll().Handler(router)
}

// Cleanup cleans up any streaming artifacts that might be left.
func Cleanup() {
	for _, s := range playbackSessions {
		s.referenceCount--
		s.CleanupIfRequired()

		if s.referenceCount > 0 {
			log.Warn("Playback session reference count leak: ", s.TranscodingSession)
		}
	}
	log.Println("Cleaned up all streaming context")
}
