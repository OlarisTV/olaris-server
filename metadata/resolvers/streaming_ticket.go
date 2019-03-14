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
	CheckCodecsPath   string
	DASHStreamingPath string
	HLSStreamingPath  string
}

// CreateSTResponseResolver resolves CreateSTResponse.
type CreateSTResponseResolver struct {
	r CreateSTResponse
}

// StreamingPath returns URI to HLS manifest.
func (r *CreateSTResponseResolver) HLSStreamingPath() string {
	return r.r.HLSStreamingPath
}

// StreamingPath returns URI to HLS manifest.
func (r *CreateSTResponseResolver) DASHStreamingPath() string {
	return r.r.DASHStreamingPath
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

	basePath := fmt.Sprintf("/s/files/jwt/%s/", token)

	checkCodecsPath := path.Join(basePath, "check-codecs.json")

	sessionID := helpers.RandAlphaString(16)
	HLSStreamingPath := path.Join(
		basePath, fmt.Sprintf("/session:%s/hls-manifest.m3u8", sessionID))
	DASHStreamingPath := path.Join(
		basePath, fmt.Sprintf("/session:%s/dash-manifest.mpd", sessionID))

	return &CreateSTResponseResolver{CreateSTResponse{
		Error:             nil,
		Jwt:               token,
		CheckCodecsPath:   checkCodecsPath,
		HLSStreamingPath:  HLSStreamingPath,
		DASHStreamingPath: DASHStreamingPath,
	}}
}
