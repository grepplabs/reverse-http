package gost

import (
	"context"
	"net"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

type chainRouteOptions struct {
	chain Chainer
}

type ChainRouteOption func(*chainRouteOptions)

type chainRoute struct {
	nodes   []*Node
	options chainRouteOptions
}

func newChainRoute(opts ...ChainRouteOption) *chainRoute {
	var options chainRouteOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return &chainRoute{
		options: options,
	}
}

func (r *chainRoute) addNode(nodes ...*Node) {
	r.nodes = append(r.nodes, nodes...)
}

func (r *chainRoute) Dial(ctx context.Context, network, address string, opts ...RouteDialOption) (net.Conn, error) {
	if len(r.Nodes()) == 0 {
		return DefaultRoute.Dial(ctx, network, address, opts...)
	}

	var options routeDialOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	conn, err := r.connect(ctx, options.logger)
	if err != nil {
		return nil, err
	}

	cc, err := r.getNode(len(r.Nodes())-1).Options().Transport.Connect(ctx, conn, network, address)
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return nil, err
	}
	return cc, nil
}

func (r *chainRoute) connect(ctx context.Context, logger *logger.Logger) (conn net.Conn, err error) {
	network := "ip"
	node := r.nodes[0]

	addr, err := Resolve(ctx, network, node.Addr, node.Options().Resolver, node.Options().HostMapper, logger)
	if err != nil {
		return
	}

	cc, err := node.Options().Transport.Dial(ctx, addr)
	if err != nil {
		return
	}

	cn := cc
	preNode := node
	for _, node := range r.nodes[1:] {
		addr, err = Resolve(ctx, network, node.Addr, node.Options().Resolver, node.Options().HostMapper, logger)
		if err != nil {
			cn.Close()
			return
		}
		cc, err = preNode.Options().Transport.Connect(ctx, cn, "tcp", addr)
		if err != nil {
			cn.Close()
			return
		}
		cn = cc
		preNode = node
	}
	conn = cn
	return
}

func (r *chainRoute) getNode(index int) *Node {
	if r == nil || len(r.Nodes()) == 0 || index < 0 || index >= len(r.Nodes()) {
		return nil
	}
	return r.nodes[index]
}

func (r *chainRoute) Nodes() []*Node {
	if r != nil {
		return r.nodes
	}
	return nil
}

func WithChainRouteChainerOption(c Chainer) ChainRouteOption {
	return func(o *chainRouteOptions) {
		o.chain = c
	}
}
