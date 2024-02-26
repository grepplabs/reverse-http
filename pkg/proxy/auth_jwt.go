package proxy

import (
	"context"
	"log/slog"

	"github.com/grepplabs/reverse-http/config"
	"github.com/grepplabs/reverse-http/pkg/gost"
	"github.com/grepplabs/reverse-http/pkg/jwtutil"
	"github.com/grepplabs/reverse-http/pkg/logger"
)

type ClientJwtAuthenticator struct {
	tokenVerifier jwtutil.TokenVerifier
	logger        *logger.Logger
}

func NewClientJwtAuthenticator(tokenVerifier jwtutil.TokenVerifier) gost.Authenticator {
	return &ClientJwtAuthenticator{
		tokenVerifier: tokenVerifier,
		logger:        logger.GetInstance(),
	}
}

func (a *ClientJwtAuthenticator) Authenticate(ctx context.Context, agentID, password string, opts ...gost.AuthOption) (id string, ok bool) {
	claims, err := a.tokenVerifier.VerifyToken(password)
	if err != nil {
		a.logger.Warn("token verification failure", slog.String("agentID", agentID), slog.String("error", err.Error()))
		return "", false
	}
	if agentID != claims.AgentID {
		a.logger.Warnf("agentID mismatch: user %s vs claim %s", agentID, claims.AgentID)
		return "", false
	}
	if config.RoleClient != claims.Role {
		a.logger.Warnf("role mismatch: role %s vs claim %s", config.RoleClient, claims.Role)
		return "", false
	}
	if agentID == "" {
		return "", false
	}
	return claims.AgentID, true
}
