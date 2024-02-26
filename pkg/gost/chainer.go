package gost

import (
	"context"
)

type ChainerOptions struct {
	Host string
}

type ChainerOption func(opts *ChainerOptions)

func ChainerHostOption(host string) ChainerOption {
	return func(opts *ChainerOptions) {
		opts.Host = host
	}
}

type Chainer interface {
	Route(ctx context.Context, network, address string, opts ...ChainerOption) Route
}
