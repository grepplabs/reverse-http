package gost

type NodeOptions struct {
	Transport  *Transport
	Resolver   Resolver
	HostMapper HostMapper
}

type NodeOption func(*NodeOptions)

type Node struct {
	Name    string
	Addr    string
	options NodeOptions
}

func NewNode(name string, addr string, opts ...NodeOption) *Node {
	var options NodeOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return &Node{
		Name:    name,
		Addr:    addr,
		options: options,
	}
}

func (node *Node) Options() *NodeOptions {
	return &node.options
}

func (node *Node) Copy() *Node {
	n := &Node{}
	*n = *node
	return n
}

func WithNodeTransport(tr *Transport) NodeOption {
	return func(o *NodeOptions) {
		o.Transport = tr
	}
}

func WithNodeResolver(resolver Resolver) NodeOption {
	return func(o *NodeOptions) {
		o.Resolver = resolver
	}
}

func WithNodeHostMapper(m HostMapper) NodeOption {
	return func(o *NodeOptions) {
		o.HostMapper = m
	}
}
