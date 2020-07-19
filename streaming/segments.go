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
	streamID := mux.Vars(r)["streamId"]
	representationId := mux.Vars(r)["representationId"]

	fileLocator, statusErr := getFileLocatorOrFail(r)
	if statusErr != nil {
		http.Error(w, statusErr.Error(), statusErr.Status())
		return
	}

	streamKey, err := getStreamKey(fileLocator, streamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims, err := getStreamingClaims(mux.Vars(r)["fileLocator"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	playbackSession, err := PBSManager.GetPlaybackSession(
		PlaybackSessionKey{
			StreamKey:        streamKey,
			sessionID:        sessionID,
			representationID: representationId,
			userID:           claims.UserID},
		InitSegmentIdx)
	defer playbackSession.Release()

	for {
		availableSegments, err := playbackSession.TranscodingSession.AvailableSegments()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if segmentPath, ok := availableSegments[ffmpeg.InitialSegmentIdx]; ok {
			log.Info("Serving path ", segmentPath, " with MIME type ", videoMIMEType)
			w.Header().Set("Content-Type", videoMIMEType)
			http.ServeFile(w, r, segmentPath)

			playbackSession.lastAccessed = time.Now()
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
	streamID := mux.Vars(r)["streamId"]

	segmentIdx, err := strconv.Atoi(mux.Vars(r)["segmentId"])
	if err != nil {
		http.Error(w, "Invalid segmentId", http.StatusBadRequest)
	}

	fileLocator, statusErr := getFileLocatorOrFail(r)
	if statusErr != nil {
		http.Error(w, statusErr.Error(), statusErr.Status())
		return
	}

	streamKey, err := getStreamKey(fileLocator, streamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims, err := getStreamingClaims(mux.Vars(r)["fileLocator"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	playbackSession, err := PBSManager.GetPlaybackSession(
		PlaybackSessionKey{
			streamKey,
			sessionID,
			representationId,
			claims.UserID,
		},
		segmentIdx)
	defer playbackSession.Release()

	for {
		availableSegments, err := playbackSession.TranscodingSession.AvailableSegments()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		segmentIdxToServe := playbackSession.lastServedSegmentIdx + 1
		if segmentPath, ok := availableSegments[segmentIdxToServe]; ok {
			log.Info("Serving path ", segmentPath, " with MIME type ", mimeType)
			w.Header().Set("Content-Type", mimeType)
			http.ServeFile(w, r, segmentPath)

			// Sometimes video.js seems to request the same segment twice, deal with that.
			if playbackSession.lastRequestedSegmentIdx != segmentIdx {
				playbackSession.lastRequestedSegmentIdx = segmentIdx
				playbackSession.lastServedSegmentIdx++
			}
			playbackSession.lastAccessed = time.Now()
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
