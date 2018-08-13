package resolvers

import (
	"context"
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/metadata/auth"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
)

type CreateSTResponse struct {
	Error         *ErrorResolver
	Jwt           string
	StreamingPath string
}
type CreateSTResponseResolver struct {
	r CreateSTResponse
}

func (r *CreateSTResponseResolver) StreamingPath() string {
	return r.r.StreamingPath
}

func (r *CreateSTResponseResolver) Jwt() string {
	return r.r.Jwt
}
func (r *CreateSTResponseResolver) Error() *ErrorResolver {
	return r.r.Error
}

func (r *Resolver) CreateStreamingTicket(ctx context.Context, args *struct{ UUID string }) *CreateSTResponseResolver {
	userID := GetUserID(ctx)
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
	StreamingPath := fmt.Sprintf("/s/jwt/%s/hls-manifest.m3u8", token)

	return &CreateSTResponseResolver{CreateSTResponse{Error: nil, Jwt: token, StreamingPath: StreamingPath}}
}
