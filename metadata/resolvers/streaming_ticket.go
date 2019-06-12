package resolvers

import (
	"context"
	"fmt"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"path"
)

// CreateSTResponse  holds new jwt data.
type CreateSTResponse struct {
	Error             *ErrorResolver
	Jwt               string
	MetadataPath      string
	DASHStreamingPath string
	HLSStreamingPath  string
	Streams           []*StreamResolver
}

// CreateSTResponseResolver resolves CreateSTResponse.
type CreateSTResponseResolver struct {
	r CreateSTResponse
}

// MetadataPath has the path to the various metadata codecs
func (r *CreateSTResponseResolver) MetadataPath() string {
	return r.r.MetadataPath
}

// HLSStreamingPath returns URI to HLS manifest.
func (r *CreateSTResponseResolver) HLSStreamingPath() string {
	return r.r.HLSStreamingPath
}

// DASHStreamingPath returns URI to DASH manifest.
func (r *CreateSTResponseResolver) DASHStreamingPath() string {
	return r.r.DASHStreamingPath
}

// Streams returns all known streams for the file in question
func (r *CreateSTResponseResolver) Streams() []*StreamResolver {
	return r.r.Streams
}

// Jwt returns streaming token.
func (r *CreateSTResponseResolver) Jwt() string {
	return r.r.Jwt
}

// Error returns error.
func (r *CreateSTResponseResolver) Error() *ErrorResolver {
	return r.r.Error
}

// CreateStreamingTicket create a new streaming request for the given content.
func (r *Resolver) CreateStreamingTicket(ctx context.Context, args *struct{ UUID string }) *CreateSTResponseResolver {
	userID, _ := auth.UserID(ctx)
	mr := db.FindContentByUUID(args.UUID)

	var filePath string
	var streamables []*StreamResolver

	if mr.Movie != nil {
		filePath = mr.Movie.FilePath
	}

	if mr.Episode != nil {
		filePath = mr.Episode.FilePath
	}

	if filePath == "" {
		return &CreateSTResponseResolver{CreateSTResponse{Error: CreateErrResolver(fmt.Errorf("No file found for UUID %s", args.UUID))}}
	}

	token, err := auth.CreateStreamingJWT(userID, filePath)
	if err != nil {
		return &CreateSTResponseResolver{CreateSTResponse{Error: CreateErrResolver(err)}}
	}

	// TODO(Maran) It would be better to somehow pass routing information along and not hard-code this in place.
	basePath := fmt.Sprintf("/olaris/s/files/jwt/%s/", token)

	metadataPath := path.Join(basePath, "metadata.json")

	sessionID := helpers.RandAlphaString(16)
	HLSStreamingPath := path.Join(
		basePath, fmt.Sprintf("/session:%s/hls-manifest.m3u8", sessionID))
	DASHStreamingPath := path.Join(
		basePath, fmt.Sprintf("/session:%s/dash-manifest.mpd", sessionID))

	var link string
	for _, stream := range mr.Movie.Streams {
		// TODO: See if we can add audio here as well
		if stream.StreamType == "subtitle" {
			link = path.Join(basePath, fmt.Sprintf("/session:%s/%d/webvtt/0.vtt", sessionID, stream.StreamId))
			streamables = append(streamables, &StreamResolver{stream, link})
		}

	}

	return &CreateSTResponseResolver{CreateSTResponse{
		Error:             nil,
		Jwt:               token,
		MetadataPath:      metadataPath,
		HLSStreamingPath:  HLSStreamingPath,
		DASHStreamingPath: DASHStreamingPath,
		Streams:           streamables,
	}}
}
