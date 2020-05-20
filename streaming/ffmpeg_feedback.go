package streaming

import (
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func serveFFmpegFeedback(w http.ResponseWriter, r *http.Request) {
	playbackSessionID := mux.Vars(r)["playbackSessionID"]

	s, err := PBSManager.GetPlaybackSessionByID(playbackSessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer s.Release()

	progressPercent, err := strconv.ParseFloat(r.URL.Query().Get("progress"), 32)
	if err == nil {
		s.TranscodingSession.ProgressPercent = float32(progressPercent)
	}

	if s.shouldThrottle() {
		s.TranscodingSession.Throttled = true
		w.Write([]byte("throttle\n"))
	} else {
		s.TranscodingSession.Throttled = false
	}
}
