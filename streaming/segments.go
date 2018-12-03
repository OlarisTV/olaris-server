package streaming

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/streaming/auth"
	"net/http"
	"strconv"
	"time"
)

var videoMIMEType = "video/mp4"

func serveInit(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["sessionID"]
	fileLocator := mux.Vars(r)["fileLocator"]
	streamID := mux.Vars(r)["streamId"]
	representationId := mux.Vars(r)["representationId"]

	ctx := r.Context()

	streamKey, err := getStreamKey(fileLocator, streamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !mediaFileURLExists(streamKey.MediaFileURL) {
		http.NotFound(w, r)
		return
	}

	claims, err := getStreamingClaims(fileLocator)
	if err == nil {
		ctx = auth.ContextWithUserIDFromStreamingClaims(ctx, claims)
	}

	// TODO(Leon Handreke): Add a special getter that will give us any of the
	// existing sessions, they all have a valid initial segment
	playbackSession, err := GetPlaybackSession(ctx, streamKey, sessionID, representationId, -1)
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
	representationId := mux.Vars(r)["representationId"]
	fileLocator := mux.Vars(r)["fileLocator"]
	streamID := mux.Vars(r)["streamId"]

	segmentIdx, err := strconv.Atoi(mux.Vars(r)["segmentId"])
	if err != nil {
		http.Error(w, "Invalid segmentId", http.StatusBadRequest)
	}

	ctx := r.Context()

	streamKey, err := getStreamKey(fileLocator, streamID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !mediaFileURLExists(streamKey.MediaFileURL) {
		http.NotFound(w, r)
		return
	}

	claims, err := getStreamingClaims(fileLocator)
	if err == nil {
		ctx = auth.ContextWithUserIDFromStreamingClaims(ctx, claims)
	}

	playbackSession, err := GetPlaybackSession(ctx, streamKey, sessionID, representationId, segmentIdx)
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
