package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"

	"github.com/grepplabs/reverse-http/config"
	"github.com/grepplabs/reverse-http/pkg/agent"
	"github.com/grepplabs/reverse-http/pkg/jwtutil"
	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/grepplabs/reverse-http/pkg/proxy"
)

type CLI struct {
	LogConfig    logger.LogConfig       `embed:"" prefix:"log."`
	Agent        config.AgentCmd        `name:"agent" cmd:"" help:"Start agent."`
	Proxy        config.ProxyCmd        `name:"proxy" cmd:"" help:"Start proxy server."`
	LoadBalancer config.LoadBalancerCmd `name:"lb" cmd:"" help:"Start load balancer."`
	Auth         config.AuthCmd         `name:"auth" cmd:"" help:"auth tools."`
	Version      struct {
		Verbose bool `short:"V" help:"Verbose."`
	} `name:"version" cmd:"" help:"Show version"`
}

func Execute() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name(os.Args[0]),
		kong.Description("HTTP reverse tunnel"),
		kong.Configuration(kong.JSON, "/etc/reverse-http/config.json", "~/.reverse-http.json"),
		kong.Configuration(kongyaml.Loader, "/etc/reverse-http/config.yaml", "~/.reverse-http.yaml"),
		kong.UsageOnError(),
	)
	logger.InitInstance(cli.LogConfig)
	switch ctx.Command() {
	case "agent":
		err := runAgent(&cli.Agent)
		ctx.FatalIfErrorf(err)
	case "proxy":
		err := runProxy(&cli.Proxy)
		ctx.FatalIfErrorf(err)
	case "lb":
		err := runLoadBalancer(&cli.LoadBalancer)
		ctx.FatalIfErrorf(err)
	case "auth key private":
		err := runAuthKeyPrivate(&cli.Auth.KeyCmd.PrivateCmd)
		ctx.FatalIfErrorf(err)
	case "auth key public":
		err := runAuthKeyPublic(&cli.Auth.KeyCmd.PublicCmd)
		ctx.FatalIfErrorf(err)
	case "auth jwt token":
		err := runAuthJwtToken(&cli.Auth.JwtCmd.TokenCmd)
		ctx.FatalIfErrorf(err)
	case "version":
		if cli.Version.Verbose {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
				"version": config.Version,
				"commit":  config.Commit,
				"date":    config.Date,
			})
		} else {
			fmt.Println(config.Version)
		}
	default:
		fmt.Println(ctx.Command())
		os.Exit(1)
	}
}

func runAgent(conf *config.AgentCmd) error {
	return agent.RunAgentClient(conf)
}

func runProxy(conf *config.ProxyCmd) error {
	proxy.RunProxyServer(conf)
	return nil
}

func runLoadBalancer(conf *config.LoadBalancerCmd) error {
	proxy.RunLoadBalancerServer(conf)
	return nil
}

func runAuthKeyPrivate(conf *config.AuthKeyPrivateCmd) error {
	return jwtutil.GeneratePrivateKey(conf)
}

func runAuthKeyPublic(conf *config.AuthKeyPublicCmd) error {
	return jwtutil.GeneratePublicKey(conf)
}

func runAuthJwtToken(conf *config.AuthJwtTokenCmd) error {
	return jwtutil.GenerateJWTToken(conf)
}
