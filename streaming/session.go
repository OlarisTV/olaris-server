package streaming

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/streaming/auth"
	"sync"
	"time"
)

const playbackSessionTimeout = 20 * time.Minute

// TODO(Leon Handreke): Currently, this is our mechanism for evicting duplicate sessions.
// Ideally, we would have a more generalized function that does garbage collection.
type PlaybackSessionKey struct {
	ffmpeg.StreamKey
	userID uint
}

type PlaybackSession struct {
	// Unique identifier, currently only used for ffmpeg feedback
	playbackSessionID string

	sessionID string

	transcodingSession *ffmpeg.TranscodingSession

	representationID string
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

	lastAccessed time.Time
}

// Read-modify-write mutex for sessions. This ensures that two parallel requests don't both create a session.
var sessionsMutex = sync.Mutex{}

var playbackSessions = map[PlaybackSessionKey]*PlaybackSession{}

func NewPlaybackSession(ctx context.Context, sessionID string, streamKey ffmpeg.StreamKey, representationID string, segmentIdx int) (*PlaybackSession, error) {
	stream, err := ffmpeg.GetStream(streamKey)
	if err != nil {
		return nil, err
	}
	streamRepresentation, err := ffmpeg.StreamRepresentationFromRepresentationId(
		stream, representationID)

	playbackSessionID := uuid.New().String()
	// TODO(Leon Handreke): Find a better way to build URLs
	feedbackURL := fmt.Sprintf("http://127.0.0.1:8080/s/ffmpeg/%s/feedback", playbackSessionID)
	if err != nil {
		return nil, fmt.Errorf("Failed to build FFmpeg feedback url: %s", err.Error())
	}

	transcodingSession, err := ffmpeg.NewTranscodingSession(streamRepresentation, segmentIdx, feedbackURL)
	if err != nil {
		return nil, err
	}

	s := &PlaybackSession{
		playbackSessionID:  playbackSessionID,
		sessionID:          sessionID,
		transcodingSession: transcodingSession,
		representationID:   representationID,
		// TODO(Leon Handreke): Make this nicer, introduce a "new" state
		lastRequestedSegmentIdx: segmentIdx - 1,
		lastServedSegmentIdx:    segmentIdx - 1,
		referenceCount:          1,
		lastAccessed:            time.Now(),
	}
	s.startTimeoutTicker()

	return s, nil
}

// GetPlaybackSession gets a playback session with the given key and for the given segment index.
// If the segment index is too far in the future, it will conclude that the user likely skipped ahead
// and start a new playback session.
// If segmentIdx == -1, any session will be returned for the given (StreamKey, representationID). This is useful to get
// a session to serve the init segment from because it doesn't matter where ffmpeg seeked to, the init segment will
// always be the same.
// The returned PlaybackSession must be released after use by calling ReleasePlaybackSession.
func GetPlaybackSession(ctx context.Context, streamKey ffmpeg.StreamKey, sessionID string, representationId string, segmentIdx int) (*PlaybackSession, error) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		// No user ID in context (probably direct file access debugging), generate a dummy.
		userID = 0
	}
	key := PlaybackSessionKey{
		userID:    userID,
		StreamKey: streamKey,
	}

	s := playbackSessions[key]
	if segmentIdx == -1 {
		if s != nil && s.representationID == representationId {
			s.referenceCount++
			return s, nil
		}
		// When starting a new session for the init segment, start at the beginning.
		segmentIdx = 0
	}

	if s == nil ||
		s.representationID != representationId ||
		s.sessionID != sessionID ||
		// This is a really crude heuristic. VideoJS will skip requesting a segment
		// if the previous segment already covers the whole duration of that segment.
		// E.g. if the playlist has 5s segment lengths but a segment is 15s long,
		// the next two won't be requested. This heuristic allows "skipping" at most
		// 4 segments.
		// TODO(Leon Handreke): Maybe do something more intelligent here by analyzing the
		// duration of the previous delivered segment?
		segmentIdx < s.lastRequestedSegmentIdx ||
		segmentIdx > s.lastRequestedSegmentIdx+5 {

		if s != nil {
			s.referenceCount--
			s.CleanupIfRequired()
		}

		s, err := NewPlaybackSession(ctx, sessionID, streamKey, representationId, segmentIdx)
		if err != nil {
			return nil, err
		}

		playbackSessions[key] = s
	}

	s = playbackSessions[key]
	s.referenceCount++

	return s, nil
}

// GetPlaybackSessionByID gets the playback session by its ID. If one with the given ID does not exist,
// and error is returned.
// The returned PlaybackSession must be released after use by calling ReleasePlaybackSession.
func GetPlaybackSessionByID(playbackSessionID string) (*PlaybackSession, error) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	var s *PlaybackSession

	for _, v := range playbackSessions {
		if v.playbackSessionID == playbackSessionID {
			s = v
			break
		}
	}

	if s == nil {
		return nil, fmt.Errorf("No PlaybackSession with the given ID %s", playbackSessionID)
	}

	s.referenceCount++
	return s, nil
}

func ReleasePlaybackSession(s *PlaybackSession) {
	s.referenceCount--
	s.CleanupIfRequired()
}

func (s *PlaybackSession) CleanupIfRequired() {
	if s.referenceCount > 0 {
		return
	}

	s.transcodingSession.Destroy()
}

func (s *PlaybackSession) shouldThrottle() bool {
	segments, _ := s.transcodingSession.AvailableSegments()

	maxSegmentIdx := -1
	for segmentIdx, _ := range segments {
		if segmentIdx > maxSegmentIdx {
			maxSegmentIdx = segmentIdx
		}
	}
	return maxSegmentIdx >= (s.lastServedSegmentIdx + 10)
}

func (s *PlaybackSession) startTimeoutTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				if time.Since(s.lastAccessed) > playbackSessionTimeout {
					s.referenceCount--
					s.CleanupIfRequired()

					ticker.Stop()
					return
				}
			}
		}
	}()
}
