package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/grepplabs/reverse-http/config"
	"github.com/grepplabs/reverse-http/pkg/agent"
	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/grepplabs/reverse-http/pkg/util"
	"github.com/quic-go/quic-go"
)

type AgentID string
type AgentDialFunc func(ctx context.Context, agentID AgentID) (net.Conn, error)

type QuicServer struct {
	conf             *config.ProxyCmd
	logger           *logger.Logger
	agentDialTimeout time.Duration
	agentVerifier    agent.Verifier
	connTrack        *ConnTrack
}

func NewQuicServer(conf *config.ProxyCmd, agentVerifier agent.Verifier, connTrack *ConnTrack, logger *logger.Logger) *QuicServer {
	return &QuicServer{
		conf:             conf,
		agentVerifier:    agentVerifier,
		agentDialTimeout: conf.AgentServer.Agent.DialTimeout,
		connTrack:        connTrack,
		logger:           logger,
	}
}
func (qs *QuicServer) Close() {
	qs.connTrack.Shutdown()
}

func (qs *QuicServer) listenForAgents(ctx context.Context, ln *quic.Listener) error {
	qs.logger.Info("waiting for agents ...")

	for {
		conn, err := ln.Accept(ctx)
		if err != nil {
			return err
		}
		go func(conn quic.Connection) {
			log := qs.logger.With(slog.String("connID", getConnID(conn)))
			log.Info(fmt.Sprintf("got a connection from: %s ", conn.RemoteAddr().String()))
			attrs, err := qs.agentVerifier.Verify(ctx, conn)
			if err != nil {
				log.Error("agent auth failure", slog.String("error", err.Error()))
				_ = conn.CloseWithError(500, err.Error())
				return
			}
			if attrs.AgentID == "" {
				log.Warn("empty agent id")
				_ = conn.CloseWithError(400, "empty agent id")
				return
			}
			agentID := AgentID(attrs.AgentID)
			log.Info(fmt.Sprintf("authenticated agent %s", agentID))
			err = qs.connTrack.PutConn(agentID, conn)
			if err != nil {
				log.Error("conn track put failed", slog.String("error", err.Error()))
				_ = conn.CloseWithError(500, "conn track put failure")
				return
			}
		}(conn)
	}
}

func (qs *QuicServer) DialAgent(ctx context.Context, agentID AgentID) (net.Conn, error) {
	conn, ok := qs.connTrack.GetConn(agentID)
	if !ok {
		return nil, fmt.Errorf("connection for agent %s not found", agentID)
	}
	if qs.agentDialTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, qs.agentDialTimeout)
		defer cancel()
	}
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &util.QuicConn{
		Stream: stream,
		LAddr:  conn.LocalAddr(),
		RAddr:  conn.RemoteAddr(),
	}, err
}
