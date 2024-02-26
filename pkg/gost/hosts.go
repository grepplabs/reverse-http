package gost

import (
	"context"
	"net"
)

type hostsOptions struct{}

type HostsOption func(opts *hostsOptions)

type HostMapper interface {
	Lookup(ctx context.Context, network, host string, opts ...HostsOption) ([]net.IP, bool)
}
