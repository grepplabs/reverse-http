package gost

import (
	"context"
	"net"
	"time"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

type RouterOptions struct {
	Retries    int
	Timeout    time.Duration
	Chain      Chainer
	Resolver   Resolver
	HostMapper HostMapper
	Logger     *logger.Logger
}

type RouterOption func(*RouterOptions)

type Router struct {
	options RouterOptions
}

func NewRouter(opts ...RouterOption) *Router {
	r := &Router{
		options: RouterOptions{
			Logger: logger.GetInstance().WithFields(map[string]any{"kind": "router"}),
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&r.options)
		}
	}
	return r
}

func (r *Router) Options() *RouterOptions {
	if r == nil {
		return nil
	}
	return &r.options
}

func (r *Router) Dial(ctx context.Context, network, address string) (conn net.Conn, err error) {
	conn, err = r.dial(ctx, network, address)
	if err != nil {
		return
	}
	return
}

func (r *Router) dial(ctx context.Context, network, address string) (conn net.Conn, err error) {
	count := r.options.Retries + 1
	if count <= 0 {
		count = 1
	}
	r.options.Logger.Debugf("dial %s/%s", address, network)

	for i := 0; i < count; i++ {
		var ipAddr string
		ipAddr, err = Resolve(ctx, "ip", address, r.options.Resolver, r.options.HostMapper, r.options.Logger)
		if err != nil {
			r.options.Logger.Error(err.Error())
			break
		}

		var route Route
		if r.options.Chain != nil {
			route = r.options.Chain.Route(ctx, network, ipAddr, ChainerHostOption(address))
		}
		if route == nil {
			route = DefaultRoute
		}
		conn, err = route.Dial(ctx, network, ipAddr,
			WithRouteDialLogger(r.options.Logger),
			WithRouteDialTimeout(r.options.Timeout),
		)
		if err == nil {
			break
		}
		r.options.Logger.Errorf("route(retry=%d) %s", i, err)
	}
	return
}

func WithRouterTimeout(timeout time.Duration) RouterOption {
	return func(o *RouterOptions) {
		o.Timeout = timeout
	}
}

func WithRouterRetriesOption(retries int) RouterOption {
	return func(o *RouterOptions) {
		o.Retries = retries
	}
}

func WithRouterChainer(chain Chainer) RouterOption {
	return func(o *RouterOptions) {
		o.Chain = chain
	}
}

func WithRouterResolver(resolver Resolver) RouterOption {
	return func(o *RouterOptions) {
		o.Resolver = resolver
	}
}

func WithRouterHostMapper(m HostMapper) RouterOption {
	return func(o *RouterOptions) {
		o.HostMapper = m
	}
}

func WithRouterLogger(logger *logger.Logger) RouterOption {
	return func(o *RouterOptions) {
		o.Logger = logger
	}
}
