package streaming

//go:generate sh -c "printf 'package streaming\n\nvar schemaTxt = `%s`\n' \"$(cat schema.graphql)\" > schematxt.go"

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

type StreamingResolver struct {
	manager PlaybackSessionManager
}

// Episode returns episode.
func (r *StreamingResolver) Sessions(ctx context.Context) (resolvers []*SessionResolver) {
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
	return t.s.FileLocator.Path
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
func (t *SessionResolver) Codecs() string {
	return t.s.TranscodingSession.Stream.Representation.Codecs
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
