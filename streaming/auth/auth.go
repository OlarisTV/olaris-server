package auth

import (
	"context"
	"fmt"
	"gitlab.com/olaris/olaris-server/metadata/auth"
)

const userIDContextKey = "auth context key userID"

func ContextWithUserIDFromStreamingClaims(ctx context.Context, claims *auth.StreamingClaims) context.Context {
	return context.WithValue(ctx, userIDContextKey, claims.UserID)
}

func UserIDFromContext(ctx context.Context) (uint, error) {
	v := ctx.Value(userIDContextKey)
	if v != nil {
		return v.(uint), nil
	}
	return 0, fmt.Errorf("No user ID in context")
}
