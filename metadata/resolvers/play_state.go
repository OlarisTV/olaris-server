package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

type PlayStateArgs struct {
	UUID     string
	Finished bool
	Playtime float64
}

type CreatePSResponseResolver struct {
	success bool
}

func (self *CreatePSResponseResolver) Success() bool {
	return self.success
}

func (r *Resolver) CreatePlayState(ctx context.Context, args *PlayStateArgs) *CreatePSResponseResolver {
	userID := GetUserID(ctx)
	ok := db.CreatePlayState(userID, args.UUID, args.Finished, args.Playtime)
	// Supply simple struct with true or false only for now
	return &CreatePSResponseResolver{success: ok}
}
