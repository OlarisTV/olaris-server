package resolvers

import (
	"context"
	"fmt"
)

func GetUserAdmin(ctx context.Context) bool {
	admin := ctx.Value("is_admin").(*bool)
	fmt.Println("ADMIN:", *admin)
	return *admin
}

func GetUserID(ctx context.Context) uint {
	id := ctx.Value("user_id").(*uint)
	fmt.Println("User:", *id)
	return *id
}
