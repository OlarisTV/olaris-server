package resolvers

import (
	"context"
	"fmt"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// CreateSTResponse  holds new jwt data.
type CreateSTResponse struct {
	Error         *ErrorResolver
	Jwt           string
	StreamingPath string
}

// CreateSTResponseResolver resolves CreateSTResponse.
type CreateSTResponseResolver struct {
	r CreateSTResponse
}

// StreamingPath returns URI to HLS manifest.
func (r *CreateSTResponseResolver) StreamingPath() string {
	return r.r.StreamingPath
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
	StreamingPath := fmt.Sprintf("/s/files/jwt/%s/hls-manifest.m3u8", token)

	return &CreateSTResponseResolver{CreateSTResponse{Error: nil, Jwt: token, StreamingPath: StreamingPath}}
}
