package gost

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

type Connector interface {
	Connect(ctx context.Context, conn net.Conn, network, address string, opts ...ConnectOption) (net.Conn, error)
}

type ConnectorAuth interface {
	Auth(ctx context.Context) *url.Userinfo
}

type connectorOptions struct {
	auth           ConnectorAuth
	tlsConfig      *tls.Config
	logger         *logger.Logger
	connectTimeout time.Duration
}

type ConnectorOption func(opts *connectorOptions)

type connectOptions struct {
	netDialer *NetDialer
}

type ConnectOption func(opts *connectOptions)

type httpConnector struct {
	options connectorOptions
}

func NewHttpConnector(opts ...ConnectorOption) Connector {
	options := connectorOptions{
		logger: logger.GetInstance().WithFields(map[string]any{"kind": "connector"}),
	}
	for _, opt := range opts {
		opt(&options)
	}

	return &httpConnector{
		options: options,
	}
}

func (c *httpConnector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...ConnectOption) (net.Conn, error) {
	log := c.options.logger.WithFields(map[string]any{
		"local":   conn.LocalAddr().String(),
		"remote":  conn.RemoteAddr().String(),
		"network": network,
		"address": address,
	})
	log.Debugf("connect %s/%s", address, network)

	req := &http.Request{
		Method:     http.MethodConnect,
		URL:        &url.URL{Host: address},
		Host:       address,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}
	req.Header.Set("Proxy-Connection", "keep-alive")

	if c.options.auth != nil {
		if user := c.options.auth.Auth(ctx); user != nil {
			u := user.Username()
			p, _ := user.Password()
			req.Header.Set("Proxy-Authorization",
				"Basic "+base64.StdEncoding.EncodeToString([]byte(u+":"+p)))
		}
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		if _, ok := conn.(net.PacketConn); ok {
			err := fmt.Errorf("tcp over udp is unsupported")
			log.Error(err.Error())
			return nil, err
		}
	default:
		err := fmt.Errorf("network %s is unsupported", network)
		log.Error(err.Error())
		return nil, err
	}

	if log.IsLevelEnabled(logger.LevelTrace) {
		dump, _ := httputil.DumpRequest(req, false)
		log.Trace(string(dump))
	}
	if c.options.connectTimeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(c.options.connectTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	req = req.WithContext(ctx)
	if err := req.Write(conn); err != nil {
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, err
	}

	if log.IsLevelEnabled(logger.LevelTrace) {
		dump, _ := httputil.DumpResponse(resp, false)
		log.Trace(string(dump))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	return conn, nil
}

func WithConnectorAuth(auth ConnectorAuth) ConnectorOption {
	return func(opts *connectorOptions) {
		opts.auth = auth
	}
}

func WithConnectorTLSConfig(tlsConfig *tls.Config) ConnectorOption {
	return func(opts *connectorOptions) {
		opts.tlsConfig = tlsConfig
	}
}

func WithConnectorLogger(logger *logger.Logger) ConnectorOption {
	return func(opts *connectorOptions) {
		opts.logger = logger
	}
}

func WithConnectorConnectTimeout(connectTimeout time.Duration) ConnectorOption {
	return func(opts *connectorOptions) {
		opts.connectTimeout = connectTimeout
	}
}

func NetDialerConnectOption(netd *NetDialer) ConnectOption {
	return func(opts *connectOptions) {
		opts.netDialer = netd
	}
}
