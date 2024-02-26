package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/grepplabs/reverse-http/config"
	"github.com/grepplabs/reverse-http/pkg/jwtutil"
	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/quic-go/quic-go"
)

type JWTAuthenticator struct {
	authFlow *authFlow
	token    string
}

func NewJWTAuthenticator(token string) (Authenticator, error) {
	if token == "" {
		return nil, errors.New("jwt auth: empty token")
	}
	return &JWTAuthenticator{
		token: token,
		authFlow: &authFlow{
			timeout: defaultTimeout,
			logger:  logger.GetInstance(),
		},
	}, nil
}

func (r *JWTAuthenticator) Authenticate(ctx context.Context, conn quic.Connection) error {
	return r.authFlow.authenticate(ctx, conn, r.token)
}

type JWTVerifier struct {
	authFlow      *authFlow
	tokenVerifier jwtutil.TokenVerifier
}

func NewJWTVerifier(tokenVerifier jwtutil.TokenVerifier) Verifier {
	return &JWTVerifier{
		authFlow: &authFlow{
			timeout: defaultTimeout,
			logger:  logger.GetInstance(),
		},
		tokenVerifier: tokenVerifier,
	}
}

func (r *JWTVerifier) Verify(ctx context.Context, conn quic.Connection) (*Attributes, error) {
	return r.authFlow.verify(ctx, conn, r.verifyToken)
}

func (r *JWTVerifier) verifyToken(token string) (*Attributes, error) {
	claims, err := r.tokenVerifier.VerifyToken(token)
	if err != nil {
		return nil, err
	}
	if claims.Role != config.RoleAgent {
		return nil, fmt.Errorf("role mismatch: role %s vs claim %s", config.RoleAgent, claims.Role)
	}
	return &Attributes{
		AgentID: claims.AgentID,
		Role:    claims.Role,
	}, nil
}
