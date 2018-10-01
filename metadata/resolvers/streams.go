package resolvers

import (
	"fmt"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

var updateStreamLock = false

// StreamResolver resolves stream information.
type StreamResolver struct {
	r db.Stream
}

// Do we really need to do all this ugly pointer stuff to let graphql handle empty values?

// CodecName returns codecname
func (r *StreamResolver) CodecName() *string {
	return &r.r.CodecName
}

// CodecMime returns codecmime
func (r *StreamResolver) CodecMime() *string {
	return &r.r.Codecs
}

// Profile returns codec profile.
func (r *StreamResolver) Profile() *string {
	return &r.r.Profile
}

// BitRate returns stream bitrate if a fixed bitrate is used.
func (r *StreamResolver) BitRate() *int32 {
	a := int32(r.r.BitRate)
	return &a
}

// StreamID returns stream id.
func (r *StreamResolver) StreamID() *int32 {
	a := int32(r.r.StreamId)
	return &a
}

// TotalDuration returns the total duriation of the stream in seconds.
func (r *StreamResolver) TotalDuration() *float64 {
	seconds := r.r.TotalDuration.Seconds()
	return &seconds
}

// StreamType returns stream type (sub/audio/video)
func (r *StreamResolver) StreamType() *string {
	return &r.r.StreamType
}

// Language returns language information for the given string if present.
func (r *StreamResolver) Language() *string {
	return &r.r.Language
}

// Title returns stream title if present.
func (r *StreamResolver) Title() *string {
	return &r.r.Title
}

// Resolution returns stream resolution if present.
func (r *StreamResolver) Resolution() *string {
	if r.r.Width != 0 {
		a := fmt.Sprintf("%dx%d", r.r.Width, r.r.Height)
		return &a
	}
	return new(string)
}

// UpdateStreams is a resolver method for the UpdateStreams method
func (r *Resolver) UpdateStreams(args *mustUUIDArgs) bool {
	if args.UUID != nil {
		ok := db.UpdateStreams(args.UUID)
		return ok
	}

	if updateStreamLock == false {
		updateStreamLock = true
		go func() {
			db.UpdateAllStreams()
			updateStreamLock = false
		}()
	}

	return true
}
