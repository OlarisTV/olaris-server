package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// UserResolver resolves user.
type UserResolver struct {
	r db.User
}

// Username returns username.
func (r *UserResolver) Username() string {
	return r.r.Username
}

// ID returns the unique UserID
func (r *UserResolver) ID() int32 {
	return int32(r.r.ID)
}

// Admin returns admin status.
func (r *UserResolver) Admin() bool {
	return r.r.Admin
}

// UserResponse holds user information and error if needed.
type UserResponse struct {
	Error *ErrorResolver
	User  *UserResolver
}

// UserResponseResolver resolves userresponse.
type UserResponseResolver struct {
	r *UserResponse
}

// Error returns error.
func (r *UserResponseResolver) Error() *ErrorResolver {
	return r.r.Error
}

// User returns user.
func (r *UserResponseResolver) User() *UserResolver {
	return r.r.User
}

// Users returns all present users.
func (r *Resolver) Users(ctx context.Context) (users []*UserResolver) {
	err := ifAdmin(ctx)
	if err == nil {
		for _, user := range db.AllUsers() {
			users = append(users, &UserResolver{user})
		}
	}
	return users
}

// DeleteUser deletes the given user.
func (r *Resolver) DeleteUser(ctx context.Context, args struct{ ID int32 }) *UserResponseResolver {
	err := ifAdmin(ctx)
	if err != nil {
		return &UserResponseResolver{&UserResponse{Error: CreateErrResolver(err)}}
	}

	user, err := db.DeleteUser(int(args.ID))

	if err != nil {
		return &UserResponseResolver{&UserResponse{Error: CreateErrResolver(err)}}
	}

	return &UserResponseResolver{&UserResponse{User: &UserResolver{user}}}

}
