package gost

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

type Listener interface {
	Init(ctx context.Context) error
	Accept() (net.Conn, error)
	Addr() net.Addr
	Close() error
}

type listenerOptions struct {
	addr      string
	logger    *logger.Logger
	tlsConfig *tls.Config
}

type ListenerOption func(opts *listenerOptions)

type tcpListener struct {
	ln      net.Listener
	logger  *logger.Logger
	options listenerOptions
}

func NewTcpListener(opts ...ListenerOption) Listener {
	options := listenerOptions{
		logger: logger.GetInstance().WithFields(map[string]any{"kind": "listener"}),
	}
	for _, opt := range opts {
		opt(&options)
	}
	return &tcpListener{
		logger:  options.logger,
		options: options,
	}
}

func (l *tcpListener) Init(ctx context.Context) (err error) {
	network := "tcp"
	lc := net.ListenConfig{}

	ln, err := lc.Listen(ctx, network, l.options.addr)
	if err != nil {
		return
	}
	if l.options.tlsConfig != nil {
		ln = tls.NewListener(ln, l.options.tlsConfig)
	}
	l.ln = ln
	return
}

func (l *tcpListener) Accept() (conn net.Conn, err error) {
	return l.ln.Accept()
}

func (l *tcpListener) Addr() net.Addr {
	return l.ln.Addr()
}

func (l *tcpListener) Close() error {
	return l.ln.Close()
}

func WithListenerAddr(addr string) ListenerOption {
	return func(opts *listenerOptions) {
		opts.addr = addr
	}
}
func WithListenerLogger(logger *logger.Logger) ListenerOption {
	return func(opts *listenerOptions) {
		opts.logger = logger
	}
}

func WithListenerTLSConfig(tlsConfig *tls.Config) ListenerOption {
	return func(opts *listenerOptions) {
		opts.tlsConfig = tlsConfig
	}
}
