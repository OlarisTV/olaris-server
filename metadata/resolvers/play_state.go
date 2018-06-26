package resolvers

import (
	"context"
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
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
	id := ctx.Value("user_id").(*uint)
	fmt.Println("user ID:", id)
	fmt.Println("Playtime", args.Playtime)
	ok := db.CreatePlayState(*id, args.UUID, args.Finished, args.Playtime)
	// Supply simple struct with true or false only for now
	return &CreatePSResponseResolver{success: ok}
}
