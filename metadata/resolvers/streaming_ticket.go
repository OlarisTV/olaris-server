package resolvers

import (
	"context"
	"fmt"
	"gitlab.com/olaris/olaris-server/filesystem"
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
}

// CreateSTResponseResolver resolves CreateSTResponse.
type CreateSTResponseResolver struct {
	r CreateSTResponse
}

func (r *CreateSTResponseResolver) MetadataPath() string {
	return r.r.MetadataPath
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
	var filePath, remote string
	var backend int

	if mr.Movie != nil {
		backend = mr.Movie.GetLibrary().Backend
		remote = mr.Movie.GetLibrary().RcloneName
		filePath = mr.Movie.FilePath
	}
	if mr.Episode != nil {
		backend = mr.Episode.GetLibrary().Backend
		remote = mr.Episode.GetLibrary().RcloneName
		filePath = mr.Episode.FilePath
	}

	if filePath == "" {
		return &CreateSTResponseResolver{CreateSTResponse{Error: CreateErrResolver(fmt.Errorf("No file found for UUID %s", args.UUID))}}
	}

	var node filesystem.Node
	var err error
	if backend == db.BackendLocal {
		node, err = filesystem.LocalNodeFromPath(filePath)
	} else if backend == db.BackendRclone {
		node, err = filesystem.RcloneNodeFromPath(path.Join(remote, filePath))
	}
	if err != nil {
		return &CreateSTResponseResolver{CreateSTResponse{Error: CreateErrResolver(err)}}
	}

	token, err := auth.CreateStreamingJWT(userID, node.FileLocator())
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

	return &CreateSTResponseResolver{CreateSTResponse{
		Error:             nil,
		Jwt:               token,
		MetadataPath:      metadataPath,
		HLSStreamingPath:  HLSStreamingPath,
		DASHStreamingPath: DASHStreamingPath,
	}}
}
