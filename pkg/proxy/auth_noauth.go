package proxy

import (
	"context"

	"github.com/grepplabs/reverse-http/pkg/gost"
)

type ClientNoAuthAuthenticator struct {
}

func NewClientNoAuthAuthenticator() gost.Authenticator {
	return &ClientNoAuthAuthenticator{}
}

func (a *ClientNoAuthAuthenticator) Authenticate(ctx context.Context, agentID, password string, opts ...gost.AuthOption) (id string, ok bool) {
	if agentID == "" {
		return "", false
	}
	return agentID, true
}
