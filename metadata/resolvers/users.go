package resolvers

import (
	"context"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
)

type UserResolver struct {
	r db.User
}

func (r *UserResolver) Login() string {
	return r.r.Login
}
func (r *UserResolver) Admin() bool {
	return r.r.Admin
}

type UserResponse struct {
	Error *ErrorResolver
	User  *UserResolver
}

type UserResponseResolver struct {
	r *UserResponse
}

func (r *UserResponseResolver) Error() *ErrorResolver {
	return r.r.Error
}
func (r *UserResponseResolver) User() *UserResolver {
	return r.r.User
}

func (r *Resolver) Users() (users []*UserResolver) {
	for _, user := range db.AllUsers() {
		users = append(users, &UserResolver{user})
	}
	return users
}

func (r *Resolver) DeleteUser(ctx context.Context, args struct{ ID int32 }) *UserResponseResolver {
	err := IfAdmin(ctx)
	if err != nil {
		return &UserResponseResolver{&UserResponse{Error: CreateErrResolver(err)}}
	}

	user, err := db.DeleteUser(int(args.ID))

	if err != nil {
		return &UserResponseResolver{&UserResponse{Error: CreateErrResolver(err)}}
	} else {
		return &UserResponseResolver{&UserResponse{User: &UserResolver{user}}}
	}

}
