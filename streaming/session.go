package streaming

import (
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"sync"
	"time"
)

type PlaybackSessionKey struct {
	ffmpeg.StreamKey
	representationID string
	sessionID        string
}

type PlaybackSession struct {
	transcodingSession *ffmpeg.TranscodingSession
	// lastRequestedSegmentIdx is the last segment index requested by the client. Some clients notice that the segments
	// we serve are actually longer than 5s and therefore skip segment indices, some will just request the next segment
	// regardless of how long the previously-loaded segment was. We have a window of max 5 (defined below), allowing
	// for segment lengths of up to 25s before our logic gets confused.
	lastRequestedSegmentIdx int
	// lastServedSegmentIdx tracks the actual index of the last segment we served, regardless of what index the client
	// requested it as. This will always increase by 1 with each subsequent segment that the client requests.
	lastServedSegmentIdx int

	// Explicit reference count to ensure that we don't destroy this session while
	// requests are still waiting for a product of this session.
	// Should be initialized to 1.
	referenceCount int
}

// Read-modify-write mutex for sessions. This ensures that two parallel requests don't both create a session.
var sessionsMutex = sync.Mutex{}

var playbackSessions = map[PlaybackSessionKey]*PlaybackSession{}

func NewPlaybackSession(key PlaybackSessionKey, segmentIdx int) (*PlaybackSession, error) {
	streamRepresentation, err := streamRepresentationFromPlaybackSessionKey(key)
	if err != nil {
		return nil, err
	}

	transcodingSession, err := ffmpeg.NewTranscodingSession(*streamRepresentation, segmentIdx)
	if err != nil {
		return nil, err
	}

	s := &PlaybackSession{
		transcodingSession: transcodingSession,
		// TODO(Leon Handreke): Make this nicer, introduce a "new" state
		lastRequestedSegmentIdx: segmentIdx - 1,
		lastServedSegmentIdx:    segmentIdx - 1,
		referenceCount:          1,
	}
	go func() {
		for range time.Tick(5000 * time.Millisecond) {
			s.throttleIfRequired()
		}
	}()

	return s, nil
}

// GetPlaybackSession gets a playback session with the given key and for the given segment index.
// If the segment index is too far in the future, it will conclude that the user likely skipped ahead
// and start a new playback session.
// If segmentIdx == -1, any session will be returned for the given key. This is useful to get a session
// to serve the init segment from.
func GetPlaybackSession(key PlaybackSessionKey, segmentIdx int) (*PlaybackSession, error) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	s := playbackSessions[key]
	if segmentIdx == -1 {
		if s != nil {
			s.referenceCount++
			return s, nil
		}
		// When starting a new session for the init segment, start at the beginning.
		segmentIdx = 0
	}

	if s == nil ||
		// This is a really crude heuristic. VideoJS will skip requesting a segment
		// if the previous segment already covers the whole duration of that segment.
		// E.g. if the playlist has 5s segment lengths but a segment is 15s long,
		// the next two won't be requested. This heuristic allows "skipping" at most
		// 4 segments.
		// TODO(Leon Handreke): Maybe do something more intelligent here by analyzing the
		// duration of the previous delivered segment?
		segmentIdx <= s.lastRequestedSegmentIdx ||
		segmentIdx > s.lastRequestedSegmentIdx+5 {

		if s != nil {
			s.referenceCount--
			s.CleanupIfRequired()
		}

		s, err := NewPlaybackSession(key, segmentIdx)
		if err != nil {
			return nil, err
		}

		playbackSessions[key] = s
	}

	s = playbackSessions[key]
	s.referenceCount++

	return s, nil
}

func ReleasePlaybackSession(s *PlaybackSession) {
	s.referenceCount--
	s.CleanupIfRequired()
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

func (s *PlaybackSession) CleanupIfRequired() {
	if s.referenceCount > 0 {
		return
	}

	s.transcodingSession.Destroy()
}

func (s *PlaybackSession) throttleIfRequired() {
	segments, _ := s.transcodingSession.AvailableSegments()

	maxSegmentIdx := -1
	for segmentIdx, _ := range segments {
		if segmentIdx > maxSegmentIdx {
			maxSegmentIdx = segmentIdx
		}
	}

	// We transcode to always be 10 segments "ahead"
	if maxSegmentIdx >= (s.lastServedSegmentIdx + 10) {
		s.transcodingSession.SetThrottled(true)
	} else {
		s.transcodingSession.SetThrottled(false)
	}
}
