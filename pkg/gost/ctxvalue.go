package gost

import (
	"context"
	"net/url"
)

type clientIDKey struct{}
type ClientID string

var (
	keyClientID = &clientIDKey{}
)

func ContextWithClientID(ctx context.Context, clientID ClientID) context.Context {
	return context.WithValue(ctx, keyClientID, clientID)
}

func ClientIDFromContext(ctx context.Context) ClientID {
	v, _ := ctx.Value(keyClientID).(ClientID)
	return v
}

type proxyAuthorizationKey struct{}

var (
	keyProxyAuthorization = &proxyAuthorizationKey{}
)

func ContextWithProxyAuthorization(ctx context.Context, auth *url.Userinfo) context.Context {
	return context.WithValue(ctx, keyProxyAuthorization, auth)
}

func ProxyAuthorizationFromContext(ctx context.Context) *url.Userinfo {
	v, _ := ctx.Value(keyProxyAuthorization).(*url.Userinfo)
	return v
}
