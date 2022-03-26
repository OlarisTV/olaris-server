package ffmpeg

// SessionState represents the current state of a TranscodingSession
type SessionState int

const (
	SessionStateNew SessionState = iota
	SessionStateRunning
	SessionStateThrottled
	SessionStateStopping
	SessionStateExited
)
