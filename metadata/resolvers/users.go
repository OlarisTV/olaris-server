package resolvers

import (
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
)

type UserResolver struct {
	r db.User
}

type CreateUserArgs struct {
	Login    string
	Password string
	Admin    bool
}

type CreateUserResponse struct {
	Error *ErrorResolver
	User  *UserResolver
}

type CreateUserResponseResolver struct {
	r *CreateUserResponse
}

func (r *CreateUserResponseResolver) Error() *ErrorResolver {
	return r.r.Error
}
func (r *CreateUserResponseResolver) User() *UserResolver {
	return r.r.User
}

func (r *Resolver) Users() (users []*UserResolver) {
	for _, user := range db.AllUsers() {
		users = append(users, &UserResolver{user})
	}
	return users
}

func (r *UserResolver) Login() string {
	return r.r.Login
}
func (r *UserResolver) Admin() bool {
	return r.r.Admin
}
