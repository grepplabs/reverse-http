package gost

import (
	"context"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

type chainOptions struct {
	logger *logger.Logger
}

type ChainOption func(*chainOptions)

type Chain struct {
	name   string
	nodes  []*Node
	logger *logger.Logger
}

func NewChain(name string, opts ...ChainOption) *Chain {
	options := chainOptions{
		logger: logger.GetInstance().WithFields(map[string]any{"kind": "chain"}),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return &Chain{
		name:   name,
		logger: options.logger,
	}
}

func (c *Chain) AddNode(node *Node) {
	if node != nil {
		c.nodes = append(c.nodes, node)
	}
}

func (c *Chain) Name() string {
	return c.name
}

func (c *Chain) Route(ctx context.Context, network, address string, opts ...ChainerOption) Route {
	if c == nil || len(c.nodes) == 0 {
		return nil
	}

	var options ChainerOptions
	for _, opt := range opts {
		opt(&options)
	}
	rt := newChainRoute(WithChainRouteChainerOption(c))
	rt.addNode(c.nodes...)
	return rt
}

func WithChainLogger(logger *logger.Logger) ChainOption {
	return func(opts *chainOptions) {
		opts.logger = logger
	}
}
