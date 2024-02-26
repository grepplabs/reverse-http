package proxy

import (
	"context"
	"net"
	"net/url"
	"time"

	certconfig "github.com/grepplabs/cert-source/config"
	tlsserverconfig "github.com/grepplabs/cert-source/tls/server/config"
	"github.com/grepplabs/reverse-http/pkg/gost"
	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/grepplabs/reverse-http/pkg/util"
)

type AgentDialer struct {
	agentDialFunc AgentDialFunc
}

func NewAgentDialer(dialAgentFunc AgentDialFunc) gost.Dialer {
	return &AgentDialer{
		agentDialFunc: dialAgentFunc,
	}
}
func (cd *AgentDialer) Dial(ctx context.Context, addr string, opts ...gost.DialerOption) (net.Conn, error) {
	agentId := gost.ClientIDFromContext(ctx)
	return cd.agentDialFunc(ctx, AgentID(agentId))
}

type HttpProxyServer struct {
	ln                  gost.Listener
	dialAgentFunc       AgentDialFunc
	clientAuthenticator gost.Authenticator
	bypass              *util.Whitelist
	forwardAuth         bool
}

func NewHttpProxyServer(listenAddr string, tlsServerConfig certconfig.TLSServerConfig, dialAgentFunc AgentDialFunc, clientAuthenticator gost.Authenticator, bypass *util.Whitelist, forwardAuth bool) (*HttpProxyServer, error) {
	listenOpts := []gost.ListenerOption{
		gost.WithListenerAddr(listenAddr),
	}
	if tlsServerConfig.Enable {
		log := logger.GetInstance().WithFields(map[string]any{"kind": "http-proxy"})
		tlsConfig, err := tlsserverconfig.GetServerTLSConfig(log.Logger, &tlsServerConfig)
		if err != nil {
			return nil, err
		}
		listenOpts = append(listenOpts, gost.WithListenerTLSConfig(tlsConfig))
	}

	ln := gost.NewTcpListener(listenOpts...)
	err := ln.Init(context.Background())
	if err != nil {
		return nil, err
	}
	return &HttpProxyServer{
		ln:                  ln,
		dialAgentFunc:       dialAgentFunc,
		clientAuthenticator: clientAuthenticator,
		bypass:              bypass,
		forwardAuth:         forwardAuth,
	}, nil
}

func (p *HttpProxyServer) Shutdown(_ context.Context) error {
	p.ln.Close()
	return nil
}

func (p *HttpProxyServer) ListenAndServe() error {
	httpProxyChain := p.getHttpProxyChain("localhost:3129", p.dialAgentFunc, p.forwardAuth)
	routerOpts := []gost.RouterOption{
		gost.WithRouterChainer(httpProxyChain),
	}
	router := gost.NewRouter(routerOpts...)
	httpHandlerOpts := []gost.HandlerOption{
		gost.WithHandlerRouter(router),
		gost.WithHandlerAuther(p.clientAuthenticator),
		gost.WithProxyOnly(), // block non-CONNECT request
	}
	if p.bypass != nil {
		httpHandlerOpts = append(httpHandlerOpts, gost.WithHandlerBypass(p.bypass))
	}
	httpHandler := gost.NewHttpHandler(httpHandlerOpts...)
	service := gost.NewService(p.ln, httpHandler)
	return service.Serve()
}

func (p *HttpProxyServer) getHttpProxyChain(addr string, dialAgentFunc AgentDialFunc, forwardAuth bool) *gost.Chain {
	c := gost.NewChain("agent-chain")

	var connectorOpts []gost.ConnectorOption
	if forwardAuth {
		connectorOpts = append(connectorOpts, gost.WithConnectorAuth(NewHttpProxyForwardAuth()))
	}
	httpConnector := gost.NewHttpConnector(connectorOpts...)
	agentDialer := NewAgentDialer(dialAgentFunc)
	tr := gost.NewTransport(agentDialer, httpConnector,
		gost.WithTransportAddr(addr),
		gost.WithTransportTimeout(10*time.Second),
	)
	nodeOpts := []gost.NodeOption{
		gost.WithNodeTransport(tr),
	}
	node := gost.NewNode("agent-node", addr, nodeOpts...)
	c.AddNode(node)
	return c
}

type HttpProxyForwardAuth struct {
}

func NewHttpProxyForwardAuth() *HttpProxyForwardAuth {
	return &HttpProxyForwardAuth{}
}

func (a *HttpProxyForwardAuth) Auth(ctx context.Context) *url.Userinfo {
	return gost.ProxyAuthorizationFromContext(ctx)
}
