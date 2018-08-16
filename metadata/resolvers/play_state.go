package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

type playStateArgs struct {
	UUID     string
	Finished bool
	Playtime float64
}

// CreatePSResponseResolver returns whether the playstate was created.
type CreatePSResponseResolver struct {
	success bool
}

// Success returns true if successfully created.
func (res *CreatePSResponseResolver) Success() bool {
	return res.success
}

// CreatePlayState creates a new playstate (or overwrite an existing one) for the given media.
func (r *Resolver) CreatePlayState(ctx context.Context, args *playStateArgs) *CreatePSResponseResolver {
	userID, _ := auth.UserID(ctx)
	ok := db.CreatePlayState(userID, args.UUID, args.Finished, args.Playtime)
	// Supply simple struct with true or false only for now
	return &CreatePSResponseResolver{success: ok}
}
