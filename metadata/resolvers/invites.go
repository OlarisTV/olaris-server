package resolvers

import (
	"context"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
)

type InviteResolver struct {
	r db.Invite
}

func (self *InviteResolver) Code() *string {
	return &self.r.Code
}

func (self *InviteResolver) User() (user *UserResolver) {
	if self.r.UserID != 0 {
		user := db.FindUser(self.r.UserID)
		return &UserResolver{user}
	}

	return nil
}

func (r *Resolver) Invites() *[]*InviteResolver {
	var invites []*InviteResolver
	for _, invite := range db.AllInvites() {
		invites = append(invites, &InviteResolver{invite})
	}
	return &invites
}

type UserInviteResponse struct {
	Error *ErrorResolver
	Code  string
}
type UserInviteResponseResolver struct {
	r *UserInviteResponse
}

func (r *UserInviteResponseResolver) Error() *ErrorResolver {
	return r.r.Error
}
func (r *UserInviteResponseResolver) Code() string {
	return r.r.Code
}

//TODO(Maran): Refactor all this error/not-error response stuff.
func (r *Resolver) CreateUserInvite(ctx context.Context) *UserInviteResponseResolver {
	err := IfAdmin(ctx)
	if err == nil {
		invite := db.CreateInvite()
		return &UserInviteResponseResolver{&UserInviteResponse{Error: nil, Code: invite.Code}}
	} else {
		return &UserInviteResponseResolver{&UserInviteResponse{Error: CreateErrResolver(err), Code: ""}}
	}
}
