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

// PlayStateResponseResolver returns whether the playstate was created.
type PlayStateResponseResolver struct {
	success   bool
	uuid      string
	playstate *PlayStateResolver
}

// Success returns true if successfully created.
func (res *PlayStateResponseResolver) Success() bool {
	return res.success
}

// UUID mediaitem
func (res *PlayStateResponseResolver) UUID() string {
	return res.uuid
}

// PlayState holds the ps object
func (res *PlayStateResponseResolver) PlayState() *PlayStateResolver {
	return res.playstate
}

// CreatePlayState creates a new playstate (or overwrite an existing one) for the given media.
func (r *Resolver) CreatePlayState(ctx context.Context, args *playStateArgs) *PlayStateResponseResolver {
	userID, _ := auth.UserID(ctx)
	ps := db.CreatePlayState(userID, args.UUID, args.Finished, args.Playtime)
	// Supply simple struct with true or false only for now
	return &PlayStateResponseResolver{success: true, uuid: ps.UUID, playstate: &PlayStateResolver{*ps}}
}

// BoolResponseResolver is a resolver with a bool success flag
type BoolResponseResolver struct {
	success bool
}

// Success resolves success
func (bs *BoolResponseResolver) Success() bool {
	return bs.success
}
