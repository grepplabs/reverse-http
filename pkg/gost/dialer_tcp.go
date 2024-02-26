package gost

import (
	"context"
	"net"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

type tcpDialerOptions struct {
	logger *logger.Logger
}

type TcpDialerOption func(opts *tcpDialerOptions)

type tcpDialer struct {
	logger *logger.Logger
}

func NewTcpDialer(opts ...TcpDialerOption) Dialer {
	options := &tcpDialerOptions{
		logger: logger.GetInstance().WithFields(map[string]any{"kind": "tcp-dialer"}),
	}
	for _, opt := range opts {
		opt(options)
	}

	return &tcpDialer{
		logger: options.logger,
	}
}

func (d *tcpDialer) Dial(ctx context.Context, addr string, opts ...DialerOption) (net.Conn, error) {
	var options DialOptions
	for _, opt := range opts {
		opt(&options)
	}
	conn, err := options.NetDialer.Dial(ctx, "tcp", addr)
	if err != nil {
		d.logger.Error(err.Error())
	}
	return conn, err
}

func WithTcpDialerLogger(logger *logger.Logger) TcpDialerOption {
	return func(opts *tcpDialerOptions) {
		opts.logger = logger
	}
}
