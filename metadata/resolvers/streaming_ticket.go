package resolvers

import (
	"context"
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/metadata/auth"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
)

type CreateSTResponse struct {
	Error  *ErrorResolver
	Ticket string
}
type CreateSTResponseResolver struct {
	r CreateSTResponse
}

func (r *CreateSTResponseResolver) StreamingPath() string {
	return r.r.Ticket
}
func (r *CreateSTResponseResolver) Error() *ErrorResolver {
	return r.r.Error
}

func (r *Resolver) CreateStreamingTicket(ctx context.Context, args *struct{ UUID string }) *CreateSTResponseResolver {
	userID := helpers.GetUserID(ctx)
	mr := db.FindContentByUUID(args.UUID)
	var filePath string

	if mr.Movie != nil {
		filePath = mr.Movie.FilePath
	}
	if mr.TvEpisode != nil {
		filePath = mr.TvEpisode.FilePath
	}

	if filePath == "" {
		return &CreateSTResponseResolver{CreateSTResponse{Error: CreateErrResolver(fmt.Errorf("No file found for UUID %s", args.UUID))}}
	}

	token, err := auth.CreateStreamingJWT(userID, filePath)
	if err != nil {
		return &CreateSTResponseResolver{CreateSTResponse{Error: CreateErrResolver(err)}}
	}
	return &CreateSTResponseResolver{CreateSTResponse{Error: nil, Ticket: token}}
}
