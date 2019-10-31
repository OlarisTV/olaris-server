package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// InviteResolver is a resolver for the invite model.
type InviteResolver struct {
	r db.Invite
}

// Code returns the invite code.
func (ir *InviteResolver) Code() *string {
	return &ir.r.Code
}

// User returns the user who redeemed this invite.
func (ir *InviteResolver) User() (*UserResolver, error) {
	if ir.r.UserID != 0 {
		user, err := db.FindUser(ir.r.UserID)
		if err != nil {
			return nil, err
		}
		return &UserResolver{*user}, nil
	}
	// Invite has not been used yet
	return nil, nil
}

// Invites returns all current invites.
func (r *Resolver) Invites(ctx context.Context) *[]*InviteResolver {
	var invites []*InviteResolver

	err := ifAdmin(ctx)
	if err == nil {
		for _, invite := range db.AllInvites() {
			invites = append(invites, &InviteResolver{invite})
		}
	}
	return &invites
}

// UserInviteResponse response when creating a new invite.
type UserInviteResponse struct {
	Error *ErrorResolver
	Code  string
}

// UserInviteResponseResolver resolver.
type UserInviteResponseResolver struct {
	r *UserInviteResponse
}

// Error returns the error for the given object.
func (r *UserInviteResponseResolver) Error() *ErrorResolver {
	return r.r.Error
}

// Code returns the given invite code.
func (r *UserInviteResponseResolver) Code() string {
	return r.r.Code
}

// CreateUserInvite creates a new invite code.
func (r *Resolver) CreateUserInvite(ctx context.Context) *UserInviteResponseResolver {
	//TODO(Maran): Refactor all this error/not-error response stuff.
	err := ifAdmin(ctx)
	if err == nil {
		invite := db.CreateInvite()
		return &UserInviteResponseResolver{&UserInviteResponse{Error: nil, Code: invite.Code}}
	}
	return &UserInviteResponseResolver{&UserInviteResponse{Error: CreateErrResolver(err), Code: ""}}
}
