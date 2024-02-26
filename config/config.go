package config

import (
	"time"

	certconfig "github.com/grepplabs/cert-source/config"
)

const (
	ReverseHttpProto       = "reverse-http-proto"
	DefaultKeepAlivePeriod = 10 * time.Second
)

const (
	AuthNoAuth = "noauth"
	AuthJWT    = "jwt"
)

const (
	StoreNone      = "none"
	StoreMemcached = "memcached"
)

const (
	RoleClient string = "client"
	RoleAgent  string = "agent"
)

const (
	TokenFromFilePrefix = "file:"
)

type ProxyCmd struct {
	AgentServer struct {
		ListenAddress string          `default:":4242" help:"Agent server listen address."`
		TLS           TLSServerConfig `embed:"" prefix:"tls."`
		Agent         struct {
			DialTimeout time.Duration `default:"10s" help:"Agent dial timeout."`
		} `embed:"" prefix:"agent."`
	} `embed:"" prefix:"agent-server."`
	HttpProxyServer struct {
		ListenAddress string                     `default:":3128" help:"HTTP proxy listen address."`
		TLS           certconfig.TLSServerConfig `embed:"" prefix:"tls."`
		HostWhitelist []string                   `placeholder:"PATTERNS" help:"List of whitelisted hosts. Empty list allows all destinations."`
	} `embed:"" prefix:"http-proxy."`
	Auth  AuthVerifier `embed:"" prefix:"auth."`
	Store struct {
		Type             string          `enum:"none,memcached" default:"none" help:"Agent access store. One of: [none, memcached]"`
		HttpProxyAddress string          `help:"Host and port for HTTP proxy access."`
		Memcached        MemcachedConfig `embed:"" prefix:"memcached."`
	} `embed:"" prefix:"store."`
}

type LoadBalancerCmd struct {
	HttpProxyServer struct {
		ListenAddress string                     `default:":3129" help:"HTTP proxy listen address."`
		TLS           certconfig.TLSServerConfig `embed:"" prefix:"tls."`
		HostWhitelist []string                   `placeholder:"PATTERNS" help:"List of whitelisted hosts. Empty list allows all destinations."`
	} `embed:"" prefix:"http-proxy."`
	HttpConnector struct {
		TLS certconfig.TLSClientConfig `embed:"" prefix:"tls."`
	} `embed:"" prefix:"http-connector."`
	Auth  AuthVerifier `embed:"" prefix:"auth."`
	Store struct {
		Type      string          `enum:"memcached" default:"memcached" help:"Agent access store. One of: [memcached]"`
		Memcached MemcachedConfig `embed:"" prefix:"memcached."`
	} `embed:"" prefix:"store."`
}

type AgentCmd struct {
	AgentClient struct {
		ServerAddress string          `default:"localhost:4242" help:"Address of the Agent server."`
		HostWhitelist []string        `placeholder:"PATTERNS" help:"List of whitelisted hosts. Empty list allows all destinations."`
		TLS           TLSClientConfig `embed:"" prefix:"tls."`
	} `embed:"" prefix:"agent-client."`
	Auth AgentAuth `embed:"" prefix:"auth."`
}

type AuthCmd struct {
	KeyCmd AuthKeyCmd `name:"key" cmd:"" help:"Key generator."`
	JwtCmd AuthJwtCmd `name:"jwt" cmd:"" help:"JWT tools."`
}

type AuthKeyCmd struct {
	PrivateCmd AuthKeyPrivateCmd `name:"private" cmd:"" help:"Generate private key."`
	PublicCmd  AuthKeyPublicCmd  `name:"public" cmd:"" help:"Generate public key."`
}

type AuthKeyPrivateCmd struct {
	Algo       string `enum:"RS256,ES256" default:"ES256" help:"Private key type. One of: [RS256, ES256]"`
	OutputFile string `name:"out" short:"o" default:"auth-key-private.pem" placeholder:"FILE" help:"Path to the generated private key file. Use '-' for stdout."`
}

type AuthKeyPublicCmd struct {
	InputFile  string `name:"in" short:"i" default:"auth-key-private.pem" placeholder:"FILE" help:"Path to the private key file. Use '-' for stdin."`
	OutputFile string `name:"out" short:"o" default:"auth-key-public.pem" placeholder:"FILE" help:"Path to the generated public key file. Use '-' for stdout."`
}

type AuthJwtCmd struct {
	TokenCmd AuthJwtTokenCmd `name:"token" cmd:"" help:"Generate jwt token."`
}

type AuthJwtTokenCmd struct {
	AgentID    string        `help:"Agent ID." required:""`
	Role       string        `enum:"client,agent" default:"client" help:"Role. One of: [client, agent]"`
	Audience   string        `help:"Audience."`
	Duration   time.Duration `default:"24h" help:"Token duration."`
	InputFile  string        `name:"in" short:"i" default:"auth-key-private.pem" placeholder:"FILE" help:"Path to the private key file. Use '-' for stdin."`
	OutputFile string        `name:"out" short:"o" default:"jwt.b64" placeholder:"FILE" help:"Path to the generated jwt token. Use '-' for stdout."`
}

type TLSServerConfig struct {
	Refresh time.Duration             `default:"0s" help:"Interval for refreshing server TLS certificates."`
	File    certconfig.TLSServerFiles `embed:"" prefix:"file."`
}

type TLSClientConfig struct {
	Refresh            time.Duration             `default:"0s" help:"Interval for refreshing client TLS certificates."`
	InsecureSkipVerify bool                      `help:"Skip TLS verification on client side."`
	File               certconfig.TLSClientFiles `embed:"" prefix:"file."`
}

type AgentAuth struct {
	Type   string `enum:"noauth,jwt" default:"noauth" help:"Authentication type. One of: [noauth, jwt]"`
	NoAuth struct {
		AgentID string `help:"Agent ID."`
	} `embed:"" prefix:"noauth."`
	JWTAuth struct {
		Token string `placeholder:"SOURCE" help:"JWT token or 'file:<filename>'"`
	} `embed:"" prefix:"jwt."`
}

type AuthVerifier struct {
	Type        string `enum:"noauth,jwt" default:"noauth" help:"Authentication verifier. One of: [noauth, jwt]"`
	JWTVerifier struct {
		PublicKey string `placeholder:"FILE" default:"auth-key-public.pem" help:"Path to the public key."`
		Audience  string `help:"JWT audience."`
	} `embed:"" prefix:"jwt."`
}

type MemcachedConfig struct {
	Address string        `default:"localhost:11211" help:"Memcached server address."`
	Timeout time.Duration `default:"1s" help:"Dial timeout."`
}
