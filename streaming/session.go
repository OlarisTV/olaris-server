package streaming

import (
	"gitlab.com/olaris/olaris-server/ffmpeg"
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

func NewPlaybackSession(sr *ffmpeg.StreamRepresentation, segmentIdx int) (*PlaybackSession, error) {
	transcodingSession, err := ffmpeg.NewTranscodingSession(*sr, segmentIdx)
	if err != nil {
		return nil, err
	}

	return &PlaybackSession{
		transcodingSession: transcodingSession,
		// TODO(Leon Handreke): Make this nicer, introduce a "new" state
		lastRequestedSegmentIdx: segmentIdx - 1,
		lastServedSegmentIdx:    segmentIdx - 1,
	}, nil
}

func GetPlaybackSession(key PlaybackSessionKey, segmentIdx int) (*PlaybackSession, error) {
	if _, ok := playbackSessions[key]; !ok {
		streamRepresentation, err := streamRepresentationFromPlaybackSessionKey(key)
		if err != nil {
			return nil, err
		}

		playbackSessions[key], err = NewPlaybackSession(streamRepresentation, segmentIdx)
		if err != nil {
			return nil, err
		}
	}

	// This is a really crude heuristic. VideoJS will skip requesting a segment
	// if the previous segment already covers the whole duration of that segment.
	// E.g. if the playlist has 5s segment lengths but a segment is 15s long,
	// the next two won't be requested. This heuristic allows "skipping" at most
	// 4 segments.
	// TODO(Leon Handreke): Maybe do something more intelligent here by analyzing the
	// duration of the previous delivered segment?
	if segmentIdx <= playbackSessions[key].lastRequestedSegmentIdx ||
		segmentIdx > playbackSessions[key].lastRequestedSegmentIdx+5 {
		playbackSessions[key].Cleanup()

		streamRepresentation, err := streamRepresentationFromPlaybackSessionKey(key)
		if err != nil {
			return nil, err
		}

		playbackSessions[key], err = NewPlaybackSession(streamRepresentation, segmentIdx)
		if err != nil {
			return nil, err
		}
		//transcodingSession, err := ffmpeg.NewTranscodingSession(*streamRepresentation, segmentIdx)
		//if err != nil {
		//	http.Error(w, err.Error(), http.StatusInternalServerError)
		//	return
		//}
		//s.transcodingSession = transcodingSession
		//playbackSession.lastServedSegmentIdx = segmentIdx - 1
	}

	return playbackSessions[key], nil

}

func streamRepresentationFromPlaybackSessionKey(key PlaybackSessionKey) (*ffmpeg.StreamRepresentation, error) {
	stream, err := ffmpeg.GetStream(key.StreamKey)
	if err != nil {
		return nil, err
	}
	streamRepresentation, err := ffmpeg.StreamRepresentationFromRepresentationId(
		stream, key.representationID)

	return &streamRepresentation, err

}
