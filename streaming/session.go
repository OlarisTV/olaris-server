package streaming

import (
	"fmt"
	"github.com/google/uuid"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"sync"
	"time"
)

const playbackSessionTimeout = 20 * time.Minute

const InitSegmentIdx = -1

type PlaybackSessionKey struct {
	ffmpeg.StreamKey

	// Unique per user playback session, but shared between the different streams of that session.
	sessionID string

	// Identifies the representation of the stream, e.g. "direct" or "preset:480-1000k-video"
	representationID string

	userID uint
}

type PlaybackSession struct {
	PlaybackSessionKey

	// Unique identifier, currently only used for ffmpeg feedback
	playbackSessionID string

	TranscodingSession *ffmpeg.TranscodingSession

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

func NewPlaybackSession(playbackSessionKey PlaybackSessionKey, segmentIdx int) (*PlaybackSession, error) {
	stream, err := ffmpeg.GetStream(playbackSessionKey.StreamKey)
	if err != nil {
		return nil, err
	}
	streamRepresentation, err := ffmpeg.StreamRepresentationFromRepresentationId(
		stream, playbackSessionKey.representationID)

	playbackSessionID := uuid.New().String()
	// TODO(Leon Handreke): Find a better way to build URLs
	feedbackURL := fmt.Sprintf(
		"http://127.0.0.1:8080/s/ffmpeg/%s/feedback", playbackSessionID)
	if err != nil {
		return nil, fmt.Errorf("Failed to build FFmpeg feedback url: %s", err.Error())
	}

	transcodingSession, err := ffmpeg.NewTranscodingSession(
		streamRepresentation, segmentIdx, feedbackURL)
	if err != nil {
		return nil, err
	}

	s := &PlaybackSession{
		PlaybackSessionKey: playbackSessionKey,

		playbackSessionID:  playbackSessionID,
		TranscodingSession: transcodingSession,
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
// If segmentIdx == InitSegmentIdx, any session will be returned for the given (StreamKey,
// representationID). This is useful to get  a session to serve the init segment from because
// it doesn't matter where ffmpeg seeked to, the init segment will
// always be the same.
// The returned PlaybackSession must be released after use by calling ReleasePlaybackSession.
func GetPlaybackSession(
	playbackSessionKey PlaybackSessionKey,
	segmentIdx int) (*PlaybackSession, error) {

	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	s := playbackSessions[playbackSessionKey]

	// If requesting the init segment, it doesn't matter where the existing session started,
	// we can just deliver it directly. Init segments are always the same, regardless of the
	// seek location passed to ffmpeg.
	if segmentIdx == InitSegmentIdx && s != nil {
		s.referenceCount++
		return s, nil
	}

	// If the request is for the next couple of segments, i.e. not seeking
	if s != nil &&
		// This is a really crude heuristic. VideoJS will skip requesting a segment
		// if the previous segment already covers the whole duration of that segment.
		// E.g. if the playlist has 5s segment lengths but a segment is 15s long,
		// the next two won't be requested. This heuristic allows "skipping" at most
		// 4 segments.
		// TODO(Leon Handreke): Maybe do something more intelligent here by analyzing the
		// duration of the previous delivered segment?
		segmentIdx > s.lastRequestedSegmentIdx &&
		segmentIdx < s.lastRequestedSegmentIdx+5 {

		s.referenceCount++
		return s, nil
	}

	// We are either seeking or no session exists yet. Destroy any existing session and
	// start a new one
	if s != nil {
		s.referenceCount--
		s.CleanupIfRequired()
	}

	// When starting a new session for the init segment, start at the beginning.
	var startAtSegmentIdx int
	if segmentIdx == InitSegmentIdx {
		startAtSegmentIdx = 0
	} else {
		startAtSegmentIdx = segmentIdx
	}

	s, err := NewPlaybackSession(playbackSessionKey, startAtSegmentIdx)
	if err != nil {
		return nil, err
	}

	playbackSessions[playbackSessionKey] = s

	s.referenceCount++
	go garbageCollectPlaybackSessions()
	return s, nil
}

func garbageCollectPlaybackSessions() {
	// Clean up streams after a user has switched representation or after they hhave started a
	// new playback session for the same stream (e.g. by reloading the page)
	type uniqueKey struct {
		ffmpeg.StreamKey
		userID uint
	}
	playbackSessionsByUniqueKey := make(map[uniqueKey][]*PlaybackSession)
	for _, s := range playbackSessions {
		k := uniqueKey{s.StreamKey, s.userID}
		playbackSessionsByUniqueKey[k] = append(playbackSessionsByUniqueKey[k], s)
	}

	// Release old playback sessions until there is only one left
	for _, sessions := range playbackSessionsByUniqueKey {
		if len(sessions) > 1 {
			newestSession := sessions[0]
			for _, s := range sessions {
				if s.lastAccessed.After(newestSession.lastAccessed) {
					newestSession = s
				}
			}
			for _, s := range sessions {
				if s != newestSession {
					removePlaybackSession(s)
				}
			}
		}
	}
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

func removePlaybackSession(s *PlaybackSession) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	delete(playbackSessions, s.PlaybackSessionKey)
	s.Release()
}

func (s *PlaybackSession) Release() {
	s.referenceCount--
	s.CleanupIfRequired()
}

func (s *PlaybackSession) CleanupIfRequired() {
	if s.referenceCount > 0 {
		return
	}

	s.TranscodingSession.Destroy()
}

// shouldThrottle returns whether the transcoding process is far enough ahead of the current
// playback state for ffmpeg to throttle down to avoid transcoding too much, wasting resources.
func (s *PlaybackSession) shouldThrottle() bool {
	segments, _ := s.TranscodingSession.AvailableSegments()

	maxSegmentIdx := -1
	for segmentIdx, _ := range segments {
		if segmentIdx > maxSegmentIdx {
			maxSegmentIdx = segmentIdx
		}
	}
	return maxSegmentIdx >= (s.lastServedSegmentIdx + 10)
}

// startTimeoutTicker starts a timer that will destroy the session if it has not been
// accessed for playbackSessionTimeout. This ensures that no ffmpeg processes linger around.
func (s *PlaybackSession) startTimeoutTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				if time.Since(s.lastAccessed) > playbackSessionTimeout {
					removePlaybackSession(s)

					ticker.Stop()
					return
				}
			}
		}
	}()
}
