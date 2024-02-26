package gost

import "context"

type Bypass interface {
	Contains(ctx context.Context, network, addr string, opts ...BypassOption) bool
}

type BypassOptions struct {
	Host string
	Path string
}

type BypassOption func(opts *BypassOptions)

func WithBypassHost(host string) BypassOption {
	return func(opts *BypassOptions) {
		opts.Host = host
	}
}

func WithBypassPath(path string) BypassOption {
	return func(opts *BypassOptions) {
		opts.Path = path
	}
}

type bypassGroup struct {
	bypasses []Bypass
}

func BypassGroup(bypasses ...Bypass) Bypass {
	return &bypassGroup{
		bypasses: bypasses,
	}
}

func (p *bypassGroup) Contains(ctx context.Context, network, addr string, opts ...BypassOption) bool {
	for _, bypass := range p.bypasses {
		if bypass != nil && bypass.Contains(ctx, network, addr, opts...) {
			return true
		}
	}
	return false
}
