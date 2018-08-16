package resolvers

import (
	"context"
	"fmt"
	"gitlab.com/olaris/olaris-server/metadata/auth"
)

func CreateNoAuthorisationError() error {
	return fmt.Errorf("You are not authorised for this action.")
}

func IfAdmin(ctx context.Context) error {
	admin, ok := auth.UserAdmin(ctx)
	if ok && admin {
		return nil
	} else {
		return CreateNoAuthorisationError()
	}
}
