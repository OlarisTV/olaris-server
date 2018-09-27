package streaming

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"net/http"
	"strconv"
	"time"
)

type PlaybackSessionKey struct {
	ffmpeg.StreamKey
	representationID string
	sessionID        string
}

type PlaybackSession struct {
	transcodingSession      *ffmpeg.TranscodingSession
	lastRequestedSegmentIdx int
	lastServedSegmentIdx    int
}

var playbackSessions = map[PlaybackSessionKey]*PlaybackSession{}

func (s *PlaybackSession) Cleanup() {
	s.transcodingSession.Destroy()
}

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

	stream, err := ffmpeg.GetStream(streamKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	streamRepresentation, err := ffmpeg.StreamRepresentationFromRepresentationId(
		stream,
		mux.Vars(r)["representationId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	playbackSessionKey := PlaybackSessionKey{
		streamKey,
		streamRepresentation.Representation.RepresentationId,
		sessionID,
	}
	if _, ok := playbackSessions[playbackSessionKey]; !ok {
		transcodingSession, err := ffmpeg.NewTranscodingSession(streamRepresentation, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		playbackSessions[playbackSessionKey] = &PlaybackSession{
			transcodingSession:      transcodingSession,
			lastServedSegmentIdx:    -1,
			lastRequestedSegmentIdx: -1,
		}
	}
	playbackSession := playbackSessions[playbackSessionKey]

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
	stream, err := ffmpeg.GetStream(streamKey)
	streamRepresentation, err := ffmpeg.StreamRepresentationFromRepresentationId(
		stream,
		mux.Vars(r)["representationId"])

	playbackSessionKey := PlaybackSessionKey{
		streamKey,
		streamRepresentation.Representation.RepresentationId,
		sessionID,
	}

	if _, ok := playbackSessions[playbackSessionKey]; !ok {
		transcodingSession, err := ffmpeg.NewTranscodingSession(streamRepresentation, segmentIdx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		playbackSessions[playbackSessionKey] = &PlaybackSession{
			transcodingSession: transcodingSession,
			// TODO(Leon Handreke): Make this nicer, introduce a "new" state
			lastRequestedSegmentIdx: segmentIdx - 1,
			lastServedSegmentIdx:    segmentIdx - 1,
		}
	}
	playbackSession := playbackSessions[playbackSessionKey]

	// This is a really crude heuristic. VideoJS will skip requesting a segment
	// if the previous segment already covers the whole duration of that segment.
	// E.g. if the playlist has 5s segment lengths but a segment is 15s long,
	// the next two won't be requested. This heuristic allows "skipping" at most
	// 4 segments.
	// TODO(Leon Handreke): Maybe do something more intelligent here by analyzing the
	// duration of the previous delivered segment?
	if segmentIdx <= playbackSession.lastRequestedSegmentIdx ||
		segmentIdx > playbackSession.lastRequestedSegmentIdx+5 {
		playbackSession.transcodingSession.Destroy()
		transcodingSession, err := ffmpeg.NewTranscodingSession(streamRepresentation, segmentIdx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		playbackSession.transcodingSession = transcodingSession
		playbackSession.lastServedSegmentIdx = segmentIdx - 1
	}

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
