package agent

import (
	"context"
	"time"

	"github.com/quic-go/quic-go"
)

const defaultTimeout = 3 * time.Second

type Authenticator interface {
	Authenticate(ctx context.Context, conn quic.Connection) error
}

type Attributes struct {
	AgentID string
	Role    string
}

type Verifier interface {
	Verify(ctx context.Context, conn quic.Connection) (*Attributes, error)
}
