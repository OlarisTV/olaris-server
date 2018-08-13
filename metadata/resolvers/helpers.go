package resolvers

import (
	"context"
	"fmt"
)

func CreateNoAuthorisationError() error {
	return fmt.Errorf("You are not authorised for this action.")
}

func IfAdmin(ctx context.Context) error {
	if GetUserAdmin(ctx) {
		return nil
	} else {
		return CreateNoAuthorisationError()
	}
}

func GetUserAdmin(ctx context.Context) bool {
	admin := ctx.Value("is_admin").(*bool)
	return *admin
}

func GetUserID(ctx context.Context) uint {
	id := ctx.Value("user_id").(*uint)
	return *id
}
