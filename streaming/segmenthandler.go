package streaming

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg"
)

var videoMIMEType = "video/mp4"
var defaultSegmentHandlerTimeout = time.Second * 20

// segmentHandler builds a http.HandlerFunc that can serve a segment.
type segmentHandler struct {
	timeout      time.Duration
	mimeType     string
	segmentIndex *int
}

// newSegmentHandler creates a new segmentHandler.
func newSegmentHandler() *segmentHandler {
	return &segmentHandler{
		timeout:  defaultSegmentHandlerTimeout,
		mimeType: videoMIMEType,
	}
}

// withTimeout modifies the segmentHandler to have a specific timeout duration.
func (sh *segmentHandler) withTimeout(timeout time.Duration) *segmentHandler {
	sh.timeout = timeout
	return sh
}

// withMimeType modifies the segmentHandler to have a specific MIME type.
func (sh *segmentHandler) withMimeType(mimeType string) *segmentHandler {
	sh.mimeType = mimeType
	return sh
}

// withSegmentIndex modifies the segmentHandler to return a specific segment
// index.
func (sh *segmentHandler) withSegmentIndex(index int) *segmentHandler {
	sh.segmentIndex = &index
	return sh
}

// toHandlerFunc builds the segmentHandler into a http.HandlerFunc
func (sh *segmentHandler) toHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := mux.Vars(r)["sessionID"]
		streamID := mux.Vars(r)["streamId"]
		representationId := mux.Vars(r)["representationId"]

		var segmentIdx int

		// If there is no segment index set on the handler, get it from the
		// request.
		var err error
		if sh.segmentIndex != nil {
			segmentIdx = *sh.segmentIndex
		} else {
			segmentIdx, err = strconv.Atoi(mux.Vars(r)["segmentId"])
			if err != nil {
				http.Error(w, "Invalid segmentId", http.StatusBadRequest)
				return
			}
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
				StreamKey:        streamKey,
				sessionID:        sessionID,
				representationID: representationId,
				userID:           claims.UserID},
			segmentIdx)
		defer playbackSession.Release()

		timeoutContext, timeoutCancel := context.WithTimeout(context.Background(), sh.timeout)
		defer timeoutCancel()

		for {
			segmentPath, err := playbackSession.TranscodingSession.FindSegmentByIndex(segmentIdx)
			if errors.Is(err, os.ErrNotExist) {
				log.WithFields(log.Fields{"segmentIDx": segmentIdx, "segmentPath": segmentPath}).Warnln("We tried to get a segment but it wasn't available yet, sleeping")
				if err := timeoutContext.Err(); err != nil {
					log.WithError(err).Debug("Timed out waiting for segment")
					http.Error(w, err.Error(), http.StatusRequestTimeout)
					return
				}

				// Make sure the session isn't throttled and try again
				playbackSession.TranscodingSession.Resume()
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", sh.mimeType)

			if playbackSession.TranscodingSession.SegmentStartIndex == 0 {
				// If the segment doesn't need to be patched, serve the file
				log.Debug("Serving path ", segmentPath, " with MIME type ", sh.mimeType)
				http.ServeFile(w, r, segmentPath)
			} else {
				// If it does need to be patched, patch it in memory and serve
				// the patched segment
				_, fileName := path.Split(segmentPath)
				patchedSegment, err := playbackSession.TranscodingSession.PatchSegment(segmentPath)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				log.Debug("Serving patched segment ", segmentPath, " with MIME type ", sh.mimeType)
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
}

// buildMediaSegmentHandlerFunc builds a segmentHandler that serves media
// segments.
func buildMediaSegmentHandlerFunc() http.HandlerFunc {
	return newSegmentHandler().toHandlerFunc()
}

// buildInitHandlerFunc builds a segmentHandler that serves init segments.
func buildInitHandlerFunc() http.HandlerFunc {
	return newSegmentHandler().
		withSegmentIndex(InitSegmentIdx).
		toHandlerFunc()
}

// buildSubtitleSegmentHandlerFunc builds a segmentHandler that serves subtitle
// segments.
func buildSubtitleSegmentHandlerFunc() http.HandlerFunc {
	return newSegmentHandler().
		withMimeType("text/vtt").
		toHandlerFunc()
}
