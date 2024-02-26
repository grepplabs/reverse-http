package gost

import (
	"context"
	"net"
)

type Dialer interface {
	Dial(ctx context.Context, addr string, opts ...DialerOption) (net.Conn, error)
}

type DialOptions struct {
	Host      string
	NetDialer *NetDialer
}

type DialerOption func(opts *DialOptions)

func WithDialerHostOption(host string) DialerOption {
	return func(opts *DialOptions) {
		opts.Host = host
	}
}

func WithDialerNetDialer(netd *NetDialer) DialerOption {
	return func(opts *DialOptions) {
		opts.NetDialer = netd
	}
}
