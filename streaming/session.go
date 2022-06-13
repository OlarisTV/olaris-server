package streaming

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg"
)

const playbackSessionTimeout = 20 * time.Minute

const InitSegmentIdx = -1

const TranscodingSegmentBuffer = 10

var PBSManager, _ = NewPlaybackSessionManager()

func NewPlaybackSessionManager() (m *PlaybackSessionManager, cleanup func()) {
	m = &PlaybackSessionManager{
		mtx:               sync.Mutex{},
		canCreateSessions: true,
		sessions:          make(map[PlaybackSessionKey]*PlaybackSession),
	}

	return m, m.CleanupSessions
}

// DestroyAll destroys all sessions and prevents new ones from being spawned.
func (m *PlaybackSessionManager) DestroyAll(ctx context.Context) {
	m.canCreateSessions = false
	numWaiting := 0
	errChannel := make(chan error)

	// Nothing to do if there are no sessions
	if len(m.sessions) == 0 {
		return
	}

	for key, session := range m.sessions {
		session := session
		key := key
		numWaiting++

		go func() {
			err := session.TranscodingSession.Destroy()
			if err != nil {
				log.WithFields(log.Fields{"SessionID": key.sessionID}).WithError(err).Error("failed to destroy transcoding session")
			}

			numWaiting--
			errChannel <- err
		}()
	}

	// Return once either all the sessions have been destroyed or the timeout is
	// reached
	for {
		select {
		case <-errChannel:
			if numWaiting == 0 {
				return
			}
		case <-ctx.Done():
			log.Warning("Reached timeout while destroying sessions. Some FFmpeg processes and/or transcode session files might be left behind.")
			return
		}
	}
}

type PlaybackSessionManager struct {
	// Read-modify-write mutex for sessions. This ensures that two parallel requests don't both create a session.
	mtx sync.Mutex

	// Set to false during shutdown to prevent new sessions from spawning
	canCreateSessions bool

	sessions map[PlaybackSessionKey]*PlaybackSession
}

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

	TranscodingSession *ffmpeg.TranscodingSession

	// lastServedSegmentIdx tracks the index of the last segment we served.
	//
	// Note: Some clients notice that the segments we serve are actually longer
	// than 5s and therefore skip segment indices, some will just request the
	// next segment regardless of how long the previously-loaded segment was. We
	// have a window of max 5 (defined below), allowing for segment lengths of
	// up to 25s before our logic gets confused.
	lastServedSegmentIdx int

	// Explicit reference count to ensure that we don't destroy this session while
	// requests are still waiting for a product of this session.
	// Should be initialized to 1.
	referenceCount int

	lastAccessed time.Time
}

func (p *PlaybackSession) Paused() bool {
	return time.Now().After(p.lastAccessed.Add(ffmpeg.SegmentDuration).Add(time.Second * 2))
}

func NewPlaybackSession(playbackSessionKey PlaybackSessionKey, segmentIdx int, m *PlaybackSessionManager) (*PlaybackSession, error) {
	if !m.canCreateSessions {
		return nil, errors.New("cannot create new playback sessions for this manager")
	}

	stream, err := ffmpeg.GetStream(playbackSessionKey.StreamKey)
	if err != nil {
		return nil, err
	}

	streamRepresentation, err := ffmpeg.StreamRepresentationFromRepresentationId(
		stream, playbackSessionKey.representationID)

	transcodingSession, err := ffmpeg.NewTranscodingSession(streamRepresentation, segmentIdx)
	if err != nil {
		return nil, err
	}

	s := &PlaybackSession{
		PlaybackSessionKey: playbackSessionKey,
		TranscodingSession: transcodingSession,
		referenceCount:     1,
		lastAccessed:       time.Now(),
	}
	s.startMaintenanceTicker(m)

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
func (m *PlaybackSessionManager) GetPlaybackSession(
	playbackSessionKey PlaybackSessionKey,
	segmentIdx int) (*PlaybackSession, error) {

	m.mtx.Lock()
	defer m.mtx.Unlock()

	s := m.sessions[playbackSessionKey]

	// If requesting the init segment, it doesn't matter where the existing session started,
	// we can just deliver it directly. Init segments are always the same, regardless of the
	// seek location passed to ffmpeg.
	if segmentIdx == InitSegmentIdx && s != nil {
		s.referenceCount++
		return s, nil
	}

	// If the request is for the next couple of segments, i.e. not seeking
	// Note: This is a really crude heuristic. VideoJS will skip requesting a
	// segment if the previous segment already covers the whole duration of that
	// segment. E.g. if the playlist has 5s segment lengths but a segment is 15s
	// long, the next two won't be requested. This heuristic allows "skipping"
	// at most 4 segments.
	// TODO(Leon Handreke): Maybe do something more intelligent here by
	// analyzing the duration of the previous delivered segment?
	if s != nil && segmentIdx >= s.TranscodingSession.SegmentStartIndex && segmentIdx < s.lastServedSegmentIdx+5 {
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

	s, err := NewPlaybackSession(playbackSessionKey, startAtSegmentIdx, m)
	if err != nil {
		return nil, err
	}

	m.sessions[playbackSessionKey] = s

	s.referenceCount++
	go m.garbageCollectPlaybackSessions()
	return s, nil
}

func (m *PlaybackSessionManager) garbageCollectPlaybackSessions() {
	// Clean up streams after a user has switched representations, or after they have started a
	// new playback session for the same stream (e.g. by reloading the page)
	type uniqueKey struct {
		ffmpeg.StreamKey
		userID uint
	}
	playbackSessionsByUniqueKey := make(map[uniqueKey][]*PlaybackSession)
	for _, s := range m.sessions {
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
					m.removePlaybackSession(s)
				}
			}
		}
	}
}

func (m *PlaybackSessionManager) removePlaybackSession(s *PlaybackSession) {
	log.WithFields(log.Fields{"file": s.PlaybackSessionKey.FileLocator, "representationID": s.PlaybackSessionKey.representationID}).Debugln("removing playback session")

	m.mtx.Lock()
	defer m.mtx.Unlock()

	delete(m.sessions, s.PlaybackSessionKey)
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

	// This could take a minute, so don't wait.
	go func() {
		err := s.TranscodingSession.Destroy()
		if err != nil {
			log.WithField("error", err).Warnln("received an error while cleaning up transcoding folder")
		}
	}()
}

// shouldThrottle returns whether the transcoding process is far enough ahead of the current
// playback state for ffmpeg to throttle down to avoid transcoding too much, wasting resources.
func (s *PlaybackSession) shouldThrottle() bool {
	// Determine what segment we need to transcode to
	transcodeToSegmentIndex := MaxInt(s.TranscodingSession.SegmentStartIndex, s.lastServedSegmentIdx) + TranscodingSegmentBuffer

	// Check to see if that segment exists
	_, err := s.TranscodingSession.FindSegmentByIndex(transcodeToSegmentIndex)

	// The segment exists, so throttle the session
	if err == nil {
		return true
	}

	// The segment we want to transcode to doesn't exist, so don't throttle yet
	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	// If there is some other unexpected error, log it and throttle the session
	log.Errorf(
		"Unexpected error trying to find segment %d: %s\n",
		transcodeToSegmentIndex,
		err.Error())
	return true
}

// TODO: We should probably close this as soon as the stream stops just to clean up after ourselves
// startMaintenanceTicker starts a timer that will destroy the session if it has not been
// accessed for playbackSessionTimeout. This ensures that no ffmpeg processes linger around.
func (s *PlaybackSession) startMaintenanceTicker(m *PlaybackSessionManager) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				if time.Since(s.lastAccessed) > playbackSessionTimeout {
					ticker.Stop()
					m.removePlaybackSession(s)
					return
				}

				if s.TranscodingSession.State == ffmpeg.SessionStateRunning && s.shouldThrottle() {
					s.TranscodingSession.Suspend()
				}
			}
		}
	}()
}

// Cleanup cleans up any streaming artifacts that might be left.
func (m *PlaybackSessionManager) CleanupSessions() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	for _, s := range m.sessions {
		s.referenceCount--
		s.CleanupIfRequired()

		if s.referenceCount > 0 {
			log.Warn("Playback session reference count leak: ", s.TranscodingSession)
		}
	}
	log.Println("Cleaned up all streaming context")
}

func (m *PlaybackSessionManager) GetPlaybackSessions() map[PlaybackSessionKey]*PlaybackSession {
	m.mtx.Lock()

	c := make(map[PlaybackSessionKey]*PlaybackSession)
	for key, session := range m.sessions {
		c[key] = session
	}

	m.mtx.Unlock()

	return c
}
