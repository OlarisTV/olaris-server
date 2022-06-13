package streaming

//go:generate sh -c "printf 'package streaming\n\nvar schemaTxt = `%s`\n' \"$(cat schema.graphql)\" > schematxt.go"

import (
	"context"
	"fmt"
	"strings"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/metadata/auth"
)

type StreamingResolver struct {
}

// CreateNoAuthorisationError returns a standard error for unauthorised requests.
func CreateNoAuthorisationError() error {
	return fmt.Errorf("you are not authorised for this action")
}

func ifAdmin(ctx context.Context) error {
	admin, ok := auth.UserAdmin(ctx)
	if ok && admin {
		return nil
	}
	return CreateNoAuthorisationError()
}

// Sessions returns current transcoding/muxing sessions
func (r *StreamingResolver) Sessions(ctx context.Context) (resolvers []*SessionResolver) {
	err := ifAdmin(ctx)
	if err != nil {
		return resolvers
	}

	sessions := make(map[string]*SessionResolver)

	for _, s := range PBSManager.GetPlaybackSessions() {
		session := sessions[s.sessionID]

		if session == nil {
			res := &SessionResolver{s: s}
			res.streams = append(res.streams, &StreamResolver{s: s})
			sessions[s.sessionID] = res
		} else {
			res := sessions[s.sessionID]
			res.streams = append(sessions[s.sessionID].streams, &StreamResolver{s: s})
		}
	}

	for _, v := range sessions {
		resolvers = append(resolvers, v)
	}

	return resolvers
}

// SeriesResolver resolvers transcoding/muxing sessions.
type SessionResolver struct {
	s       *PlaybackSession
	streams []*StreamResolver
}

func (t *SessionResolver) FileLocator() string {
	return t.s.FileLocator.String()
}
func (t *SessionResolver) SessionID() string {
	return t.s.sessionID
}
func (t *SessionResolver) UserID() int32 {
	return int32(t.s.userID)
}
func (t *SessionResolver) Streams() []*StreamResolver {
	return t.streams
}
func (t *SessionResolver) Paused() bool {
	return t.s.Paused()
}
func (t *SessionResolver) Progress() int32 {
	return t.s.Progress()
}

type StreamResolver struct {
	s *PlaybackSession
}

// InitSchema inits the graphql schema.
func InitSchema() *graphql.Schema {
	schema := graphql.MustParseSchema(schemaTxt, &StreamingResolver{})
	return schema
}

// NewRelayHandler handles graphql requests.
func NewRelayHandler() (*graphql.Schema, *relay.Handler) {
	schema := InitSchema()
	handler := &relay.Handler{Schema: schema}
	return schema, handler
}

func (t *StreamResolver) LastAccessed() string {
	return t.s.lastAccessed.String()
}

func (t *StreamResolver) TranscodingPercentage() int32 {
	return t.s.TranscodingSession.ProgressPercentage()
}
func (t *StreamResolver) StreamID() int32 {
	return int32(t.s.StreamId)
}

func (t *StreamResolver) Throttled() bool {
	return t.s.TranscodingSession.State == ffmpeg.SessionStateThrottled
}
func (t *StreamResolver) TranscodingState() string {
	return strings.ToUpper(ffmpeg.StateToString[t.s.TranscodingSession.State])
}
func (t *StreamResolver) Transcoded() bool {
	return t.s.TranscodingSession.Stream.Representation.Transcoded
}

func (t *StreamResolver) Transmuxed() bool {
	return t.s.TranscodingSession.Stream.Representation.Transmuxed
}
func (t *StreamResolver) BitRate() int32 {
	return int32(t.s.TranscodingSession.Stream.Representation.BitRate)
}
func (t *StreamResolver) Container() string {
	return t.s.TranscodingSession.Stream.Representation.Container
}
func (t *StreamResolver) CodecName() string {
	return t.s.TranscodingSession.Stream.Stream.CodecName
}
func (t *StreamResolver) Codecs() string {
	return t.s.TranscodingSession.Stream.Stream.Codecs
}
func (t *StreamResolver) StreamType() string {
	return t.s.TranscodingSession.Stream.Stream.StreamType
}
func (t *StreamResolver) Language() string {
	return t.s.TranscodingSession.Stream.Stream.Language
}
func (t *StreamResolver) Title() string {
	return t.s.TranscodingSession.Stream.Stream.Title
}
func (t *StreamResolver) Resolution() string {
	return fmt.Sprintf("%vx%v", t.s.TranscodingSession.Stream.Representation.Width, t.s.TranscodingSession.Stream.Representation.Height)
}
