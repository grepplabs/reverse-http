package gost

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

var (
	DefaultTlsDialer = &TlsDialer{}
)

type TlsDialer struct {
	Timeout       time.Duration
	Logger        *logger.Logger
	TLSConfigFunc func() *tls.Config
}

func (d *TlsDialer) Dial(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	if d == nil {
		d = DefaultTlsDialer
	}
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	netd := net.Dialer{
		Timeout: timeout,
	}
	var tlsConfig *tls.Config

	if d.TLSConfigFunc != nil {
		tlsConfig = d.TLSConfigFunc()
	}
	if tlsConfig == nil {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}
	tlsd := tls.Dialer{
		NetDialer: &netd,
		Config:    tlsConfig,
	}
	conn, err = tlsd.DialContext(ctx, network, addr)
	if err != nil {
		log := d.Logger
		if log == nil {
			log = logger.GetInstance().WithFields(map[string]any{"kind": "tls-dialer"})
		}
		log.Debugf("dial %s failed: %s", network, err)
		return nil, err
	}
	return conn, err
}
