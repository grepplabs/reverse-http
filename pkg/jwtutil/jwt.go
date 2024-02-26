package jwtutil

import (
	"crypto"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/grepplabs/reverse-http/config"
)

type TokenAlg string

const (
	RS256 TokenAlg = "RS256"
	ES256 TokenAlg = "ES256"
)

type Role string

const (
	RoleClient Role = Role(config.RoleClient)
	RoleAgent  Role = Role(config.RoleAgent)
)

const DefaultTokenDuration = 30 * 24 * time.Hour

type TokenClaims struct {
	AgentID string `json:"agent_id"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

type TokenSignerOption func(*tokenSigner)

func WithTokenDuration(duration time.Duration) func(*tokenSigner) {
	return func(s *tokenSigner) {
		s.duration = duration
	}
}
func WithRole(role Role) func(*tokenSigner) {
	return func(s *tokenSigner) {
		s.role = role
	}
}

func WithSignerAudience(audience string) func(*tokenSigner) {
	return func(s *tokenSigner) {
		s.audience = audience
	}
}

type tokenSigner struct {
	privateKey crypto.PrivateKey
	alg        TokenAlg
	agentID    string
	duration   time.Duration
	audience   string
	role       Role
}

type TokenSigner interface {
	SignToken() (string, error)
}

func NewTokenSigner(alg TokenAlg, privateKey crypto.PrivateKey, agentID string, opts ...TokenSignerOption) TokenSigner {
	t := &tokenSigner{
		alg:        alg,
		agentID:    agentID,
		privateKey: privateKey,
		role:       RoleClient,
		duration:   DefaultTokenDuration,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (s tokenSigner) SignToken() (string, error) {
	method := jwt.GetSigningMethod(string(s.alg))
	if method == nil {
		return "", fmt.Errorf("unknown signing method: %s", s.alg)
	}
	if s.agentID == "" {
		return "", errors.New("agentID is empty")
	}
	now := time.Now()
	claims := TokenClaims{
		AgentID: s.agentID,
		Role:    string(s.role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "reverse-http",
			Subject:   s.agentID,
			ID:        uuid.New().String(),
		},
	}
	if s.audience != "" {
		claims.RegisteredClaims.Audience = []string{s.audience}
	}
	token := jwt.NewWithClaims(method, claims)
	return token.SignedString(s.privateKey)
}

type TokenVerifierOption func(*tokenVerifier)

func WithVerifierAudience(audience string) func(*tokenVerifier) {
	return func(s *tokenVerifier) {
		s.audience = audience
	}
}

type tokenVerifier struct {
	publicKey crypto.PublicKey
	audience  string
}

type TokenVerifier interface {
	VerifyToken(tokenString string) (*TokenClaims, error)
}

func NewTokenVerifier(publicKey crypto.PublicKey, opts ...TokenVerifierOption) TokenVerifier {
	t := &tokenVerifier{
		publicKey: publicKey,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (s tokenVerifier) VerifyToken(tokenString string) (*TokenClaims, error) {
	keyFunc := func(*jwt.Token) (interface{}, error) {
		return s.publicKey, nil
	}
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{string(RS256), string(ES256)}),
		jwt.WithLeeway(5 * time.Second),
	}
	if s.audience != "" {
		opts = append(opts, jwt.WithAudience(s.audience))
	}
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, keyFunc, opts...)
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*TokenClaims); ok {
		return claims, nil
	}
	return nil, errors.New("unknown claims type")
}
