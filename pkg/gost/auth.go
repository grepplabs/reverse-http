package gost

import "context"

type authOptions struct{}

type AuthOption func(opts *authOptions)

type Authenticator interface {
	Authenticate(ctx context.Context, user, password string, opts ...AuthOption) (id string, ok bool)
}
