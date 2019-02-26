package streaming

import (
	"github.com/gorilla/mux"
	"net/http"
)

func serveFFmpegFeedback(w http.ResponseWriter, r *http.Request) {
	playbackSessionID := mux.Vars(r)["playbackSessionID"]

	s, err := GetPlaybackSessionByID(playbackSessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer s.Release()

	if s.shouldThrottle() {
		w.Write([]byte("throttle\n"))
	}
}
