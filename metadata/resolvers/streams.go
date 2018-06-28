package resolvers

import (
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
)

type StreamResolver struct {
	r db.Stream
}

// Do we really need to do all this ugly pointer stuff to let graphql handle empty values?
func (r *StreamResolver) CodecName() *string {
	return &r.r.CodecName
}
func (r *StreamResolver) CodecMime() *string {
	return &r.r.Codecs
}
func (r *StreamResolver) Profile() *string {
	return &r.r.Profile
}
func (r *StreamResolver) BitRate() *int32 {
	a := int32(r.r.BitRate)
	return &a
}
func (r *StreamResolver) StreamID() *int32 {
	a := int32(r.r.StreamId)
	return &a
}
func (r *StreamResolver) StreamType() *string {
	return &r.r.StreamType
}
func (r *StreamResolver) Language() *string {
	return &r.r.Language
}
func (r *StreamResolver) Title() *string {
	return &r.r.Title
}
func (r *StreamResolver) Resolution() *string {
	if r.r.Width != 0 {
		a := fmt.Sprintf("%dx%d", r.r.Width, r.r.Height)
		return &a
	}
	return new(string)
}
