package helpers

import (
	"context"
)

func GetUserID(ctx context.Context) uint {
	id := ctx.Value("user_id").(*uint)
	return *id
}
