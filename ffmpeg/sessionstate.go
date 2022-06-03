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

var StateToString = map[SessionState]string{
	SessionStateNew:       "New",
	SessionStateRunning:   "Running",
	SessionStateThrottled: "Throttled",
	SessionStateStopping:  "Stopping",
	SessionStateExited:    "Exited",
}
