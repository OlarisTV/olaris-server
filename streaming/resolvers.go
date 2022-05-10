package streaming

//go:generate sh -c "printf 'package streaming\n\nvar schemaTxt = `%s`\n' \"$(cat schema.graphql)\" > schematxt.go"

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
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

// Episode returns episode.
func (r *StreamingResolver) Sessions(ctx context.Context) (resolvers []*SessionResolver) {
	err := ifAdmin(ctx)
	if err != nil {
		return resolvers
	}

	for _, s := range PBSManager.GetPlaybackSessions() {
		resolvers = append(resolvers, &SessionResolver{s: s})
	}

	return resolvers
}

// SeriesResolver resolvers a serie.
type SessionResolver struct {
	s *PlaybackSession
}

func (t *SessionResolver) FileLocator() string {
	return t.s.FileLocator.String()
}
func (t *SessionResolver) SessionID() string {
	return t.s.sessionID
}
func (t *SessionResolver) LastAccessed() string {
	return t.s.lastAccessed.String()
}
func (t *SessionResolver) PlaybackSessionID() string {
	return t.s.playbackSessionID
}
func (t *SessionResolver) UserID() int32 {
	return int32(t.s.userID)
}
func (t *SessionResolver) LastRequestedSegmentIdx() int32 {
	return int32(t.s.lastRequestedSegmentIdx)
}
func (t *SessionResolver) TranscodingPercentage() int32 {
	return int32(t.s.TranscodingSession.ProgressPercent)
}
func (t *SessionResolver) Throttled() bool {
	return t.s.TranscodingSession.Throttled
}
func (t *SessionResolver) Transcoded() bool {
	return t.s.TranscodingSession.Stream.Representation.Transcoded
}
func (t *SessionResolver) Transmuxed() bool {
	return t.s.TranscodingSession.Stream.Representation.Transmuxed
}
func (t *SessionResolver) BitRate() int32 {
	return int32(t.s.TranscodingSession.Stream.Representation.BitRate)
}
func (t *SessionResolver) Container() string {
	return t.s.TranscodingSession.Stream.Representation.Container
}
func (t *SessionResolver) CodecName() string {
	return t.s.TranscodingSession.Stream.Stream.CodecName
}
func (t *SessionResolver) Codecs() string {
	return t.s.TranscodingSession.Stream.Stream.Codecs
}
func (t *SessionResolver) StreamType() string {
	return t.s.TranscodingSession.Stream.Stream.StreamType
}
func (t *SessionResolver) Language() string {
	return t.s.TranscodingSession.Stream.Stream.Language
}
func (t *SessionResolver) Title() string {
	return t.s.TranscodingSession.Stream.Stream.Title
}
func (t *SessionResolver) Resolution() string {
	return fmt.Sprintf("%vx%v", t.s.TranscodingSession.Stream.Representation.Width, t.s.TranscodingSession.Stream.Representation.Height)
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
