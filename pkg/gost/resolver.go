package gost

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

var (
	ErrInvalidResolver = errors.New("invalid resolver")
)

type resolverOptions struct{}

type ResolverOption func(opts *resolverOptions)

type Resolver interface {
	Resolve(ctx context.Context, network, host string, opts ...ResolverOption) ([]net.IP, error)
}

func Resolve(ctx context.Context, network, addr string, r Resolver, hosts HostMapper, log *logger.Logger) (string, error) {
	if addr == "" {
		return addr, nil
	}

	host, port, _ := net.SplitHostPort(addr)
	if host == "" {
		return addr, nil
	}

	if hosts != nil {
		if ips, _ := hosts.Lookup(ctx, network, host); len(ips) > 0 {
			log.Debugf("hit host mapper: %s -> %s", host, ips)
			return net.JoinHostPort(ips[0].String(), port), nil
		}
	}

	if r != nil {
		ips, err := r.Resolve(ctx, network, host)
		if err != nil {
			if errors.Is(err, ErrInvalidResolver) {
				return addr, nil
			}
			log.Error(err.Error())
		}
		if len(ips) == 0 {
			return "", fmt.Errorf("resolver: domain %s does not exist", host)
		}
		return net.JoinHostPort(ips[0].String(), port), nil
	}
	return addr, nil
}
