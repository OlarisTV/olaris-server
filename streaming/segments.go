package streaming

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"net/http"
	"strconv"
	"time"
)

var videoMIMEType = "video/mp4"

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
	// TODO(Leon Handreke): Add a special getter that will give us any of the
	// existing sessions, they all have a valid initial segment
	playbackSession, err := GetPlaybackSession(playbackSessionKey, -1)
	defer ReleasePlaybackSession(playbackSession)

	for {
		availableSegments, err := playbackSession.transcodingSession.AvailableSegments()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if segmentPath, ok := availableSegments[ffmpeg.InitialSegmentIdx]; ok {
			log.Info("Serving path ", segmentPath, " with MIME type ", videoMIMEType)
			w.Header().Set("Content-Type", videoMIMEType)
			http.ServeFile(w, r, segmentPath)
			return
		} else {
			time.Sleep(100 * time.Millisecond)
			continue
		}
	}
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
	defer ReleasePlaybackSession(playbackSession)

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
	serveSegment(w, r, videoMIMEType)
}

func serveSubtitleSegment(w http.ResponseWriter, r *http.Request) {
	serveSegment(w, r, "text/vtt")
}
