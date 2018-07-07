package streaming

import (
	"github.com/gorilla/mux"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"net/http"
	"strconv"
	"time"
)

func serveInit(w http.ResponseWriter, r *http.Request) {
	streamKey, err := getStreamKey(
		mux.Vars(r)["fileLocator"],
		mux.Vars(r)["streamId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	session, err := getOrStartTranscodingSession(streamRepresentation, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, session.InitialSegment())
}

func serveSegment(w http.ResponseWriter, r *http.Request) {
	segmentId, err := strconv.Atoi(mux.Vars(r)["segmentId"])
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
	stream, err := ffmpeg.GetStream(streamKey)
	streamRepresentation, err := ffmpeg.StreamRepresentationFromRepresentationId(
		stream,
		mux.Vars(r)["representationId"])
	session, _ := getOrStartTranscodingSession(streamRepresentation, segmentId)

	segmentPath, err := session.GetSegment(segmentId, 20*time.Second)
	http.ServeFile(w, r, segmentPath)
}

func getSessions(streamKey ffmpeg.StreamKey, representationId string) []*ffmpeg.TranscodingSession {
	matching := []*ffmpeg.TranscodingSession{}

	for _, s := range sessions {
		if s.Stream.Stream.StreamKey == streamKey && s.Stream.Representation.RepresentationId == representationId {
			matching = append(matching, s)
		}
	}
	return matching
}

func getOrStartTranscodingSession(stream ffmpeg.StreamRepresentation, segmentId int) (*ffmpeg.TranscodingSession, error) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	var s *ffmpeg.TranscodingSession

	representationId := stream.Representation.RepresentationId

	matchingSessions := getSessions(stream.Stream.StreamKey, representationId)
	for _, matchingSession := range matchingSessions {
		if matchingSession.IsProjectedAvailable(segmentId, 20*time.Second) {
			s = matchingSession
			break
		}
	}

	if s == nil {
		var err error
		s, err = ffmpeg.NewTranscodingSession(stream, segmentId)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
		s.Start()
	}

	return s, nil
}
