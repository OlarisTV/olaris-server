package streaming

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

func serveInit(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["sessionID"]
	streamKey, err := getStreamKey(
		mux.Vars(r)["fileLocator"],
		mux.Vars(r)["streamId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !mediaFileURLExists(streamKey.MediaFileURL) {
		http.NotFound(w, r)
		return
	}

	playbackSessionKey := PlaybackSessionKey{
		streamKey,
		mux.Vars(r)["representationId"],
		sessionID,
	}
	playbackSession, err := GetPlaybackSession(playbackSessionKey, 0)

	http.ServeFile(w, r, playbackSession.transcodingSession.InitialSegment())
}

func serveSegment(w http.ResponseWriter, r *http.Request, mimeType string) {
	sessionID := mux.Vars(r)["sessionID"]
	segmentIdx, err := strconv.Atoi(mux.Vars(r)["segmentId"])
	if err != nil {
		http.Error(w, "Invalid segmentId", http.StatusBadRequest)
	}

	streamKey, err := getStreamKey(
		mux.Vars(r)["fileLocator"],
		mux.Vars(r)["streamId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !mediaFileURLExists(streamKey.MediaFileURL) {
		http.NotFound(w, r)
		return
	}

	playbackSessionKey := PlaybackSessionKey{
		streamKey,
		mux.Vars(r)["representationId"],
		sessionID,
	}

	playbackSession, err := GetPlaybackSession(playbackSessionKey, segmentIdx)

	for {
		availableSegments, err := playbackSession.transcodingSession.AvailableSegments()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		segmentIdxToServe := playbackSession.lastServedSegmentIdx + 1
		if segmentPath, ok := availableSegments[segmentIdxToServe]; ok {
			log.Info("Serving path ", segmentPath, " with MIME type ", mimeType)
			w.Header().Set("Content-Type", mimeType)
			http.ServeFile(w, r, segmentPath)
			playbackSession.lastRequestedSegmentIdx = segmentIdx
			playbackSession.lastServedSegmentIdx++
			return
		} else {
			time.Sleep(100 * time.Millisecond)
			continue
		}

	}
}

func serveMediaSegment(w http.ResponseWriter, r *http.Request) {
	serveSegment(w, r, "video/mp4")
}

func serveSubtitleSegment(w http.ResponseWriter, r *http.Request) {
	serveSegment(w, r, "text/vtt")
}
