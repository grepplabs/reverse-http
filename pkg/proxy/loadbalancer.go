package proxy

import (
	"context"
	"fmt"
	"net"

	tlsclient "github.com/grepplabs/cert-source/tls/client"
	"github.com/grepplabs/reverse-http/pkg/gost"
	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/grepplabs/reverse-http/pkg/store"
)

type LoadBalancerDialer struct {
	tcpDialer   gost.Dialer
	storeClient store.Client
	netDialer   *gost.NetDialer
}

func NewLoadBalancerDialer(storeClient store.Client, tlsConfigFunc tlsclient.TLSClientConfigFunc) *LoadBalancerDialer {
	return &LoadBalancerDialer{
		tcpDialer:   gost.NewTcpDialer(),
		storeClient: storeClient,
		netDialer:   newLoadBalancerNetNetDialer(tlsConfigFunc),
	}
}

func (lb *LoadBalancerDialer) Dial(ctx context.Context, agentID AgentID) (net.Conn, error) {
	addr, err := lb.storeClient.Get(string(agentID))
	if err != nil {
		return nil, err
	}
	if addr == "" {
		return nil, fmt.Errorf("target addr for agentID %s is empty", agentID)
	}
	return lb.tcpDialer.Dial(ctx, addr, gost.WithDialerNetDialer(lb.netDialer))
}

func newLoadBalancerNetNetDialer(tlsConfigFunc tlsclient.TLSClientConfigFunc) *gost.NetDialer {
	if tlsConfigFunc != nil {
		tlsDialer := gost.TlsDialer{
			Timeout:       gost.DefaultTimeout,
			Logger:        logger.GetInstance().WithFields(map[string]any{"kind": "tls-dialer"}),
			TLSConfigFunc: tlsConfigFunc,
		}
		return &gost.NetDialer{
			DialFunc: tlsDialer.Dial,
		}
	} else {
		return nil
	}
}
