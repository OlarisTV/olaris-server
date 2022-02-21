package streaming

import (
	"errors"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"net/http"
	"os"
	"path"
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
		segmentPath, err := playbackSession.TranscodingSession.FindSegmentByIndex(InitSegmentIdx)
		if errors.Is(err, os.ErrNotExist) {
			// REVIEW: Should there be a context timeout here to prevent an
			// endless loop if the segment requested will never exist?
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Serving path ", segmentPath, " with MIME type ", videoMIMEType)
		w.Header().Set("Content-Type", videoMIMEType)
		http.ServeFile(w, r, segmentPath)

		playbackSession.lastAccessed = time.Now()
		playbackSession.lastServedSegmentIdx = InitSegmentIdx
		return
	}
}

func serveSegment(w http.ResponseWriter, r *http.Request, mimeType string) {
	sessionID := mux.Vars(r)["sessionID"]
	representationId := mux.Vars(r)["representationId"]
	streamID := mux.Vars(r)["streamId"]

	segmentIdx, err := strconv.Atoi(mux.Vars(r)["segmentId"])
	if err != nil {
		http.Error(w, "Invalid segmentId", http.StatusBadRequest)
		return
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
		segmentPath, err := playbackSession.TranscodingSession.FindSegmentByIndex(segmentIdx)
		if errors.Is(err, os.ErrNotExist) {
			// Make sure the session isn't throttled and try again
			playbackSession.TranscodingSession.Resume()
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", mimeType)

		if playbackSession.TranscodingSession.SegmentStartIndex == 0 {
			// If the segment doesn't need to be patched, serve the file
			log.Info("Serving path ", segmentPath, " with MIME type ", mimeType)
			http.ServeFile(w, r, segmentPath)
		} else {
			// If it does need to be patched, patch it in memory and serve that
			_, fileName := path.Split(segmentPath)
			patchedSegment, err := playbackSession.TranscodingSession.PatchSegment(segmentPath)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Info("Serving patched segment ", segmentPath, " with MIME type ", mimeType)
			http.ServeContent(w, r, fileName, time.Now(), patchedSegment)
		}

		playbackSession.lastAccessed = time.Now()
		playbackSession.lastServedSegmentIdx = segmentIdx

		// Resume transcoding if we need to after serving this segment
		if playbackSession.TranscodingSession.State == ffmpeg.SessionStateThrottled && !playbackSession.shouldThrottle() {
			playbackSession.TranscodingSession.Resume()
		}

		return
	}
}

func serveMediaSegment(w http.ResponseWriter, r *http.Request) {
	serveSegment(w, r, videoMIMEType)
}

func serveSubtitleSegment(w http.ResponseWriter, r *http.Request) {
	serveSegment(w, r, "text/vtt")
}
