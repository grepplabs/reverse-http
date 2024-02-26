package agent

import (
	"context"
	"errors"

	"github.com/grepplabs/reverse-http/config"
	"github.com/quic-go/quic-go"
)

type NoAuthAuthenticator struct {
	authFlow *authFlow
	agentID  string
}

func NewNoAuthAuthenticator(agentID string) (Authenticator, error) {
	if agentID == "" {
		return nil, errors.New("noauth: empty agent-id")
	}
	return &NoAuthAuthenticator{
		agentID:  agentID,
		authFlow: &authFlow{timeout: defaultTimeout},
	}, nil
}

func (r *NoAuthAuthenticator) Authenticate(ctx context.Context, conn quic.Connection) error {
	return r.authFlow.authenticate(ctx, conn, r.agentID)
}

type NoAuthVerifier struct {
	authFlow *authFlow
}

func NewNoAuthVerifier() Verifier {
	return &NoAuthVerifier{
		authFlow: &authFlow{timeout: defaultTimeout},
	}
}

func (r *NoAuthVerifier) Verify(ctx context.Context, conn quic.Connection) (*Attributes, error) {
	return r.authFlow.verify(ctx, conn, r.verifyToken)
}

func (r *NoAuthVerifier) verifyToken(agentID string) (*Attributes, error) {
	return &Attributes{
		AgentID: agentID,
		Role:    config.RoleAgent,
	}, nil
}
