package gost

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/grepplabs/reverse-http/pkg/logger"
)

const (
	defaultRealm = "reverse-http"
)

type Handler interface {
	Handle(context.Context, net.Conn, ...HandleOption) error
}

type handlerOptions struct {
	logger *logger.Logger

	bypass    Bypass
	router    *Router
	auth      *url.Userinfo
	auther    Authenticator
	tlsConfig *tls.Config
	proxyOnly bool
}

type HandlerOption func(opts *handlerOptions)

type httpHandler struct {
	router  *Router
	options handlerOptions
}

func NewHttpHandler(opts ...HandlerOption) Handler {
	options := handlerOptions{
		logger: logger.GetInstance().WithFields(map[string]any{"kind": "handler"}),
	}
	for _, opt := range opts {
		opt(&options)
	}
	h := &httpHandler{
		options: options,
	}
	h.router = h.options.router
	if h.router == nil {
		h.router = NewRouter()
	}
	return h
}

func (h *httpHandler) Handle(ctx context.Context, conn net.Conn, opts ...HandleOption) error {
	defer conn.Close()
	start := time.Now()

	log := h.options.logger.WithFields(map[string]any{
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
	})
	log.Infof("handle http %s -> %s", conn.RemoteAddr(), conn.LocalAddr())
	defer func() {
		log.With(slog.Duration("duration", time.Since(start))).
			Infof("handle http %s <- %s", conn.RemoteAddr(), conn.LocalAddr())
	}()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		log.Error(err.Error())
		return err
	}
	defer req.Body.Close()

	return h.handleRequest(ctx, conn, req, log)
}

func (h *httpHandler) Close() error {
	return nil
}

func (h *httpHandler) handleRequest(ctx context.Context, conn net.Conn, req *http.Request, log *logger.Logger) error {
	if h.options.proxyOnly && req.Method != http.MethodConnect {
		resp := &http.Response{
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     http.Header{},
			StatusCode: http.StatusMethodNotAllowed,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf("Non-proxy request '%s' is not supported", req.Method))),
		}
		return resp.Write(conn)
	}

	if !req.URL.IsAbs() && govalidator.IsDNSName(req.Host) {
		req.URL.Scheme = "http"
	}
	network := "tcp"
	addr := req.Host
	if _, port, _ := net.SplitHostPort(addr); port == "" {
		addr = net.JoinHostPort(addr, "80")
	}

	fields := map[string]any{
		"dst": addr,
	}
	if u, p, _ := h.basicProxyAuth(req.Header.Get("Proxy-Authorization"), log); u != "" {
		fields["user"] = u
		ctx = ContextWithProxyAuthorization(ctx, url.UserPassword(u, p))
	}
	log = log.WithFields(fields)

	if log.IsLevelEnabled(logger.LevelTrace) {
		dump, _ := httputil.DumpRequest(req, false)
		log.Trace(string(dump))
	}
	log.Debugf("%s >> %s", conn.RemoteAddr(), addr)

	resp := &http.Response{
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	clientID, ok := h.authenticate(ctx, conn, req, resp, log)
	if !ok {
		return nil
	}
	ctx = ContextWithClientID(ctx, ClientID(clientID))

	if h.options.bypass != nil && h.options.bypass.Contains(ctx, network, addr) {
		resp.StatusCode = http.StatusForbidden

		if log.IsLevelEnabled(logger.LevelTrace) {
			dump, _ := httputil.DumpResponse(resp, false)
			log.Trace(string(dump))
		}
		log.Debugf("bypass: %s", addr)

		return resp.Write(conn)
	}

	if req.Method == "PRI" ||
		(req.Method != http.MethodConnect && req.URL.Scheme != "http") {
		resp.StatusCode = http.StatusBadRequest

		if log.IsLevelEnabled(logger.LevelTrace) {
			dump, _ := httputil.DumpResponse(resp, false)
			log.Trace(string(dump))
		}

		return resp.Write(conn)
	}

	req.Header.Del("Proxy-Authorization")

	cc, err := h.router.Dial(ctx, network, addr)
	if err != nil {
		resp.StatusCode = http.StatusServiceUnavailable

		if log.IsLevelEnabled(logger.LevelTrace) {
			dump, _ := httputil.DumpResponse(resp, false)
			log.Trace(string(dump))
		}
		_ = resp.Write(conn)
		return err
	}
	defer cc.Close()

	if req.Method == http.MethodConnect {
		resp.StatusCode = http.StatusOK
		resp.Status = "200 Connection established"

		if log.IsLevelEnabled(logger.LevelTrace) {
			dump, _ := httputil.DumpResponse(resp, false)
			log.Trace(string(dump))
		}
		if err = resp.Write(conn); err != nil {
			log.Error(err.Error())
			return err
		}
	} else {
		req.Header.Del("Proxy-Connection")
		if err = req.Write(cc); err != nil {
			log.Error(err.Error())
			return err
		}
	}

	start := time.Now()
	log.Infof("%s -> %s", conn.RemoteAddr(), addr)
	_ = NetTransport(conn, cc)
	log.WithFields(map[string]any{
		"duration": time.Since(start),
	}).Infof("%s <- %s", conn.RemoteAddr(), addr)

	return nil
}

func (h *httpHandler) basicProxyAuth(proxyAuth string, _ *logger.Logger) (username, password string, ok bool) {
	if proxyAuth == "" {
		return
	}

	if !strings.HasPrefix(proxyAuth, "Basic ") {
		return
	}
	c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(proxyAuth, "Basic "))
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}

	return cs[:s], cs[s+1:], true
}

func (h *httpHandler) authenticate(ctx context.Context, conn net.Conn, req *http.Request, resp *http.Response, log *logger.Logger) (id string, ok bool) {
	u, p, _ := h.basicProxyAuth(req.Header.Get("Proxy-Authorization"), log)
	if h.options.auther == nil {
		return "", true
	}
	if id, ok = h.options.auther.Authenticate(ctx, u, p); ok {
		return
	}
	if resp.Header == nil {
		resp.Header = http.Header{}
	}
	if resp.StatusCode == 0 {
		realm := defaultRealm
		resp.StatusCode = http.StatusProxyAuthRequired
		resp.Header.Add("Proxy-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", realm))
		if strings.ToLower(req.Header.Get("Proxy-Connection")) == "keep-alive" {
			resp.Header.Set("Connection", "close")
			resp.Header.Set("Proxy-Connection", "close")
		}

		log.Debug("proxy authentication required")
	} else {
		if resp.StatusCode == http.StatusOK {
			resp.Header.Set("Connection", "keep-alive")
		}
	}
	if log.IsLevelEnabled(logger.LevelTrace) {
		dump, _ := httputil.DumpResponse(resp, false)
		log.Trace(string(dump))
	}

	_ = resp.Write(conn)
	return
}

func WithHandlerBypass(bypass Bypass) HandlerOption {
	return func(opts *handlerOptions) {
		opts.bypass = bypass
	}
}

func WithHandlerRouter(router *Router) HandlerOption {
	return func(opts *handlerOptions) {
		opts.router = router
	}
}

func WithHandlerAuth(auth *url.Userinfo) HandlerOption {
	return func(opts *handlerOptions) {
		opts.auth = auth
	}
}

func WithHandlerAuther(auther Authenticator) HandlerOption {
	return func(opts *handlerOptions) {
		opts.auther = auther
	}
}

func WithHandlerTLSConfig(tlsConfig *tls.Config) HandlerOption {
	return func(opts *handlerOptions) {
		opts.tlsConfig = tlsConfig
	}
}

func WithHandlerLogger(logger *logger.Logger) HandlerOption {
	return func(opts *handlerOptions) {
		opts.logger = logger
	}
}

func WithProxyOnly() HandlerOption {
	return func(opts *handlerOptions) {
		opts.proxyOnly = true
	}
}

type HandleOptions struct {
}

type HandleOption func(opts *HandleOptions)
