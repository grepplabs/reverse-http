package gost

import (
	"context"
	"net"
	"time"
)

type TransportOptions struct {
	Addr    string
	Route   Route
	Timeout time.Duration
}

type TransportOption func(*TransportOptions)

type Transport struct {
	dialer    Dialer
	connector Connector
	options   TransportOptions
}

func NewTransport(d Dialer, c Connector, opts ...TransportOption) *Transport {
	tr := &Transport{
		dialer:    d,
		connector: c,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&tr.options)
		}
	}
	return tr
}

func (tr *Transport) Dial(ctx context.Context, addr string) (net.Conn, error) {
	netd := &NetDialer{
		Timeout: tr.options.Timeout,
	}
	if tr.options.Route != nil && len(tr.options.Route.Nodes()) > 0 {
		netd.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return tr.options.Route.Dial(ctx, network, addr)
		}
	}
	opts := []DialerOption{
		WithDialerHostOption(tr.options.Addr),
		WithDialerNetDialer(netd),
	}
	return tr.dialer.Dial(ctx, addr, opts...)
}

func (tr *Transport) Connect(ctx context.Context, conn net.Conn, network, address string) (net.Conn, error) {
	netd := &NetDialer{
		Timeout: tr.options.Timeout,
	}
	return tr.connector.Connect(ctx, conn, network, address,
		NetDialerConnectOption(netd),
	)
}

func (tr *Transport) Options() *TransportOptions {
	if tr != nil {
		return &tr.options
	}
	return nil
}

func (tr *Transport) Copy() *Transport {
	tr2 := &Transport{}
	*tr2 = *tr
	return tr
}

func WithTransportAddr(addr string) TransportOption {
	return func(o *TransportOptions) {
		o.Addr = addr
	}
}

func WithTransportRoute(route Route) TransportOption {
	return func(o *TransportOptions) {
		o.Route = route
	}
}

func WithTransportTimeout(timeout time.Duration) TransportOption {
	return func(o *TransportOptions) {
		o.Timeout = timeout
	}
}
