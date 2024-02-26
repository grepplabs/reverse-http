package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	tlsconfig "github.com/grepplabs/cert-source/config"
	tlsclient "github.com/grepplabs/cert-source/tls/client"
	tlsclientconfig "github.com/grepplabs/cert-source/tls/client/config"
	"github.com/grepplabs/reverse-http/config"
	"github.com/grepplabs/reverse-http/pkg/gost"
	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/grepplabs/reverse-http/pkg/util"
	"github.com/oklog/run"
	"github.com/quic-go/quic-go"
)

func RunAgentClient(conf *config.AgentCmd) error {
	log := logger.GetInstance().WithFields(map[string]any{"kind": "agent"})
	group := new(run.Group)

	util.AddQuitSignal(group)
	addAgentClient(conf, group)

	err := group.Run()
	if err != nil {
		log.Error("client exiting", slog.String("error", err.Error()))
	} else {
		log.Info("client exiting")
	}
	return nil
}

func addAgentClient(conf *config.AgentCmd, group *run.Group) {
	log := logger.GetInstance().WithFields(map[string]any{"kind": "agent"})
	ctx, cancel := context.WithCancel(context.Background())
	group.Add(func() error {
		log.Info("starting quick client")
		authenticator, err := getAuthenticator(conf)
		if err != nil {
			return err
		}
		client, err := NewQuickClient(ctx, conf.AgentClient.ServerAddress, authenticator, log, conf.AgentClient.HostWhitelist, conf.AgentClient.TLS)
		if err != nil {
			return err
		}
		client.keepConnected()
		return nil
	}, func(error) {
		cancel()
	})
}

func getAuthenticator(conf *config.AgentCmd) (Authenticator, error) {
	switch conf.Auth.Type {
	case config.AuthNoAuth:
		return NewNoAuthAuthenticator(conf.Auth.NoAuth.AgentID)
	case config.AuthJWT:
		token, err := getJWTToken(conf)
		if err != nil {
			return nil, fmt.Errorf("get jwt token failed: %v", err)
		}
		return NewJWTAuthenticator(token)
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", conf.Auth.Type)
	}
}

func getJWTToken(conf *config.AgentCmd) (string, error) {
	token := conf.Auth.JWTAuth.Token

	if strings.HasPrefix(token, config.TokenFromFilePrefix) {
		filename := strings.TrimLeft(token, config.TokenFromFilePrefix)
		content, err := os.ReadFile(filename)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
	return token, nil
}

type QuickClient struct {
	parent        context.Context
	address       string
	proxyHandler  gost.Handler
	authenticator Authenticator
	logger        *logger.Logger
	tlsConfigFunc tlsclient.TLSClientConfigFunc
}

func NewQuickClient(parent context.Context, address string, authenticator Authenticator, logger *logger.Logger, whitelist []string, tlsClientConfig config.TLSClientConfig) (*QuickClient, error) {
	tlsConfigFunc, err := tlsclientconfig.GetTLSClientConfigFunc(logger.Logger, &tlsconfig.TLSClientConfig{
		Enable:             true,
		Refresh:            tlsClientConfig.Refresh,
		InsecureSkipVerify: tlsClientConfig.InsecureSkipVerify,
		File:               tlsClientConfig.File,
	}, tlsclient.WithTLSClientNextProtos([]string{config.ReverseHttpProto}))
	if err != nil {
		return nil, err
	}
	return &QuickClient{
		parent:        parent,
		address:       address,
		proxyHandler:  httpProxyHandler(util.WhitelistFromStrings(whitelist)),
		authenticator: authenticator,
		logger:        logger,
		tlsConfigFunc: tlsConfigFunc,
	}, nil
}

func (c *QuickClient) keepConnected() {
	err := c.connectForHttpProxy()
	if err != nil {
		c.logger.Error("agent dial: " + err.Error())
	}
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-c.parent.Done():
			c.logger.Debug("context closed")
			return
		case <-ticker.C:
			err = c.connectForHttpProxy()
			if err != nil {
				c.logger.Error("agent dial: " + err.Error())
			}
		}
	}
}

func (c *QuickClient) connectForHttpProxy() error {
	c.logger.Info("connecting to " + c.address)

	tlsConf := c.tlsConfigFunc()
	conn, err := quic.DialAddr(c.parent, c.address, tlsConf, &quic.Config{
		KeepAlivePeriod: config.DefaultKeepAlivePeriod,
	})
	if err != nil {
		return err
	}
	defer conn.CloseWithError(0, "client connection closed")

	c.logger.Info("sending authenticate")
	err = c.authenticator.Authenticate(c.parent, conn)
	if err != nil {
		return err
	}
	for {
		c.logger.Info("waiting for clients")
		stream, err := conn.AcceptStream(c.parent)
		if err != nil {
			return fmt.Errorf("stream accept failure: %v", err)
		}
		log := c.logger.With(slog.Int64("stream", int64(stream.StreamID())))
		log.Info("stream accepted")

		go func() {
			defer func() {
				_ = stream.Close()
				log.Info("stream closed")
			}()

			err := c.proxyHandler.Handle(c.parent, &util.QuicConn{
				Stream: stream,
				LAddr:  conn.LocalAddr(),
				RAddr:  conn.RemoteAddr(),
			})
			if err != nil {
				log.Error("serve conn failure", slog.String("error", err.Error()))
			}
		}()
	}
}

func httpProxyHandler(bypass *util.Whitelist) gost.Handler {
	router := gost.NewRouter()
	httpHandlerOpts := []gost.HandlerOption{
		gost.WithHandlerRouter(router),
	}
	if bypass != nil {
		httpHandlerOpts = append(httpHandlerOpts, gost.WithHandlerBypass(bypass))
	}
	return gost.NewHttpHandler(httpHandlerOpts...)
}
