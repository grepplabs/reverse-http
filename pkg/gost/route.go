package gost

import (
	"context"
	"net"
	"time"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

var (
	DefaultRoute Route = &defaultRoute{}
)

type Route interface {
	Dial(ctx context.Context, network, address string, opts ...RouteDialOption) (net.Conn, error)
	Nodes() []*Node
}

type routeDialOptions struct {
	timeout time.Duration
	logger  *logger.Logger
}

type RouteDialOption func(opts *routeDialOptions)

type defaultRoute struct{}

func (*defaultRoute) Dial(ctx context.Context, network, address string, opts ...RouteDialOption) (net.Conn, error) {
	options := routeDialOptions{
		logger: logger.GetInstance().WithFields(map[string]any{"kind": "route"}),
	}
	for _, opt := range opts {
		opt(&options)
	}
	netd := NetDialer{
		Timeout: options.timeout,
		Logger:  options.logger,
	}
	return netd.Dial(ctx, network, address)
}

func (r *defaultRoute) Nodes() []*Node {
	return nil
}

func WithRouteDialTimeout(d time.Duration) RouteDialOption {
	return func(opts *routeDialOptions) {
		opts.timeout = d
	}
}
func WithRouteDialLogger(logger *logger.Logger) RouteDialOption {
	return func(opts *routeDialOptions) {
		opts.logger = logger
	}
}
