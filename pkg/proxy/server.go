package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	tlsconfig "github.com/grepplabs/cert-source/config"
	tlsclientconfig "github.com/grepplabs/cert-source/tls/client/config"
	"github.com/grepplabs/cert-source/tls/keyutil"
	tlsserver "github.com/grepplabs/cert-source/tls/server"
	tlsserverconfig "github.com/grepplabs/cert-source/tls/server/config"
	"github.com/grepplabs/reverse-http/config"
	"github.com/grepplabs/reverse-http/pkg/agent"
	"github.com/grepplabs/reverse-http/pkg/gost"
	"github.com/grepplabs/reverse-http/pkg/jwtutil"
	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/grepplabs/reverse-http/pkg/store"
	storememcached "github.com/grepplabs/reverse-http/pkg/store/memcached"
	storenone "github.com/grepplabs/reverse-http/pkg/store/none"
	"github.com/grepplabs/reverse-http/pkg/util"
	"github.com/oklog/run"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
)

func RunProxyServer(conf *config.ProxyCmd) {
	log := logger.GetInstance().WithFields(map[string]any{"kind": "proxy"})
	group := new(run.Group)

	util.AddQuitSignal(group)
	dialAgentFunc := addQuicServer(conf, group)
	addProxyHttpServer(conf, group, dialAgentFunc)

	err := group.Run()
	if err != nil {
		log.Error("server exiting", slog.String("error", err.Error()))
	} else {
		log.Info("server exiting")
	}
}

func RunLoadBalancerServer(conf *config.LoadBalancerCmd) {
	log := logger.GetInstance().WithFields(map[string]any{"kind": "lb-server"})
	group := new(run.Group)

	util.AddQuitSignal(group)
	addLoadBalancerServer(conf, group)

	err := group.Run()
	if err != nil {
		log.Error("server exiting", slog.String("error", err.Error()))
	} else {
		log.Info("server exiting")
	}
}

func Tracer(connTrack *ConnTrack) func(_ context.Context, perspective logging.Perspective, connID quic.ConnectionID) *logging.ConnectionTracer {
	return func(_ context.Context, perspective logging.Perspective, connID quic.ConnectionID) *logging.ConnectionTracer {
		ct := logging.ConnectionTracer{
			StartedConnection: func(local, remote net.Addr, srcConnID, destConnID logging.ConnectionID) {
				connTrack.logger.Info("connection start", slog.String("connID", connID.String()))
				if perspective == logging.PerspectiveServer && connID.Len() != 0 {
					connTrack.OnConnStarted(connID.String())
				}
			},
			Close: func() {
				connTrack.logger.Info("connection close", slog.String("connID", connID.String()))
				if perspective == logging.PerspectiveServer && connID.Len() != 0 {
					connTrack.OnConnClose(connID.String())
				}
			},
		}
		return logging.NewMultiplexedConnectionTracer(&ct)
	}
}

func addQuicServer(conf *config.ProxyCmd, group *run.Group) AgentDialFunc {
	log := logger.GetInstance().WithFields(map[string]any{"kind": "quic-server"})
	tlsConfig, err := tlsserverconfig.GetServerTLSConfig(log.Logger, &tlsconfig.TLSServerConfig{
		Enable:  true,
		Refresh: conf.AgentServer.TLS.Refresh,
		File:    conf.AgentServer.TLS.File,
	}, tlsserver.WithTLSServerNextProtos([]string{config.ReverseHttpProto}))
	if err != nil {
		log.Error("error while during server tls config setup", slog.String("error", err.Error()))
		os.Exit(1)
	}
	agentVerifier, err := getAgentVerifier(&conf.Auth)
	if err != nil {
		log.Error("error while agent verifier setup", slog.String("error", err.Error()))
		os.Exit(1)
	}
	storeClient, err := getProxyStoreClient(conf, log)
	if err != nil {
		log.Error("error while store client setup", slog.String("error", err.Error()))
		os.Exit(1)
	}
	httpProxyAddress := conf.Store.HttpProxyAddress
	if httpProxyAddress == "" {
		httpProxyAddress = conf.HttpProxyServer.ListenAddress
	}
	log.Info(fmt.Sprintf("store http proxy address %s", httpProxyAddress))
	connTrack := NewConnTrack(storeClient, httpProxyAddress)
	listenAddr := conf.AgentServer.ListenAddress
	log.Info(fmt.Sprintf("starting UDP agent server on %s", listenAddr))
	ln, err := quic.ListenAddr(listenAddr, tlsConfig, &quic.Config{
		KeepAlivePeriod: config.DefaultKeepAlivePeriod,
		Tracer:          Tracer(connTrack),
	})
	if err != nil {
		log.Error("error while starting agent server", slog.String("error", err.Error()))
		os.Exit(1)
	}
	quicServer := NewQuicServer(conf, agentVerifier, connTrack, log)
	group.Add(func() error {
		return quicServer.listenForAgents(context.Background(), ln)
	}, func(error) {
		log.Info("shutdown agent server ...")
		quicServer.Close()
		storeClient.Close()
		_ = ln.Close()
	})
	return quicServer.DialAgent
}

func getProxyStoreClient(conf *config.ProxyCmd, log *logger.Logger) (store.Client, error) {
	switch conf.Store.Type {
	case config.StoreNone:
		return storenone.NewClient(), nil
	case config.StoreMemcached:
		log.Infof("memcached server %s", conf.Store.Memcached.Address)
		return storememcached.NewClient(conf.Store.Memcached), nil
	default:
		return nil, fmt.Errorf("unsupported store type: %s", conf.Store.Type)
	}
}

func getAgentVerifier(conf *config.AuthVerifier) (agent.Verifier, error) {
	switch conf.Type {
	case config.AuthNoAuth:
		return agent.NewNoAuthVerifier(), nil
	case config.AuthJWT:
		publicKey, err := keyutil.ReadPublicKeyFile(conf.JWTVerifier.PublicKey)
		if err != nil {
			return nil, err
		}
		tokenVerifier := jwtutil.NewTokenVerifier(publicKey, jwtutil.WithVerifierAudience(conf.JWTVerifier.Audience))
		return agent.NewJWTVerifier(tokenVerifier), nil
	default:
		return nil, fmt.Errorf("unsupported agent verifier type: %s", conf.Type)
	}
}

func getClientVerifier(conf *config.AuthVerifier) (gost.Authenticator, error) {
	switch conf.Type {
	case config.AuthNoAuth:
		return NewClientNoAuthAuthenticator(), nil
	case config.AuthJWT:
		publicKey, err := keyutil.ReadPublicKeyFile(conf.JWTVerifier.PublicKey)
		if err != nil {
			return nil, err
		}
		tokenVerifier := jwtutil.NewTokenVerifier(publicKey, jwtutil.WithVerifierAudience(conf.JWTVerifier.Audience))
		return NewClientJwtAuthenticator(tokenVerifier), nil
	default:
		return nil, fmt.Errorf("unsupported client verifier type: %s", conf.Type)
	}
}

func addProxyHttpServer(conf *config.ProxyCmd, group *run.Group, dialAgentFunc AgentDialFunc) {
	log := logger.GetInstance().WithFields(map[string]any{"kind": "http-proxy"})
	clientVerifier, err := getClientVerifier(&conf.Auth)
	if err != nil {
		log.Error("error while client verifier setup", slog.String("error", err.Error()))
		os.Exit(1)
	}
	const forwardAuth = false
	listenAddr := conf.HttpProxyServer.ListenAddress
	srv, err := NewHttpProxyServer(listenAddr, conf.HttpProxyServer.TLS, dialAgentFunc, clientVerifier, util.WhitelistFromStrings(conf.HttpProxyServer.HostWhitelist), forwardAuth)
	if err != nil {
		log.Error("error while starting http proxy server", slog.String("error", err.Error()))
		os.Exit(1)
	}
	group.Add(func() error {
		log.Infof("starting TCP http proxy server on %s", listenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}, func(error) {
		log.Info("shutdown http proxy server ...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("server http proxy shutdown", slog.String("error", err.Error()))
		}
	})
}

func addLoadBalancerServer(conf *config.LoadBalancerCmd, group *run.Group) {
	log := logger.GetInstance().WithFields(map[string]any{"kind": "lb-server"})
	clientVerifier, err := getClientVerifier(&conf.Auth)
	if err != nil {
		log.Error("error while client verifier setup", slog.String("error", err.Error()))
		os.Exit(1)
	}
	storeClient, err := getLoadBalancerStoreClient(conf, log)
	if err != nil {
		log.Error("error while store client setup", slog.String("error", err.Error()))
		os.Exit(1)
	}

	tlsConfigFunc, err := tlsclientconfig.GetTLSClientConfigFunc(log.Logger, &conf.HttpConnector.TLS)
	if err != nil {
		log.Error("error while connector tls setup", slog.String("error", err.Error()))
		os.Exit(1)
	}
	const forwardAuth = true
	dialAgentFunc := NewLoadBalancerDialer(storeClient, tlsConfigFunc)
	listenAddr := conf.HttpProxyServer.ListenAddress
	srv, err := NewHttpProxyServer(listenAddr, conf.HttpProxyServer.TLS, dialAgentFunc.Dial, clientVerifier, util.WhitelistFromStrings(conf.HttpProxyServer.HostWhitelist), forwardAuth)
	if err != nil {
		log.Error("error while starting lb proxy server", slog.String("error", err.Error()))
		os.Exit(1)
	}
	group.Add(func() error {
		log.Infof("starting TCP lb proxy server on %s", listenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}, func(error) {
		log.Info("shutdown lb proxy server ...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("server lb proxy shutdown", slog.String("error", err.Error()))
		}
	})
}

func getLoadBalancerStoreClient(conf *config.LoadBalancerCmd, log *logger.Logger) (store.Client, error) {
	switch conf.Store.Type {
	case config.StoreMemcached:
		log.Infof("memcached server %s", conf.Store.Memcached.Address)
		return storememcached.NewClient(conf.Store.Memcached), nil
	default:
		return nil, fmt.Errorf("unsupported store type: %s", conf.Store.Type)
	}
}
