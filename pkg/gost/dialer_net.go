package gost

import (
	"context"
	"net"
	"time"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

const (
	DefaultTimeout = 10 * time.Second
)

var (
	DefaultNetDialer = &NetDialer{}
)

type NetDialer struct {
	Timeout  time.Duration
	DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)
	Logger   *logger.Logger
}

func (d *NetDialer) Dial(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	if d == nil {
		d = DefaultNetDialer
	}
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	if d.DialFunc != nil {
		return d.DialFunc(ctx, network, addr)
	}
	netd := net.Dialer{
		Timeout: timeout,
	}
	conn, err = netd.DialContext(ctx, network, addr)
	if err != nil {
		log := d.Logger
		if log == nil {
			log = logger.GetInstance().WithFields(map[string]any{"kind": "net-dialer"})
		}
		log.Debugf("dial %s failed: %s", network, err)
		return nil, err
	}
	return conn, err
}
