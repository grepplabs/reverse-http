package jwtutil

import (
	"crypto"
	"testing"
	"time"

	"github.com/grepplabs/cert-source/tls/keyutil"
	"github.com/stretchr/testify/require"
)

func TestRSASignAndVerify(t *testing.T) {
	privKey, _, pubKey, _, err := keyutil.GenerateRSAKeys()
	require.NoError(t, err)
	_, _, pubKeyOther, _, err := keyutil.GenerateRSAKeys()
	require.NoError(t, err)
	testSignAndVerify(t, RS256, privKey, pubKey, pubKeyOther)
}

func TestECSignAndVerify(t *testing.T) {
	privKey, _, pubKey, _, err := keyutil.GenerateECKeys()
	require.NoError(t, err)
	_, _, pubKeyOther, _, err := keyutil.GenerateECKeys()
	require.NoError(t, err)
	testSignAndVerify(t, ES256, privKey, pubKey, pubKeyOther)
}

func testSignAndVerify(t *testing.T, alg TokenAlg, privKey crypto.PrivateKey, pubKey crypto.PublicKey, pubKeyOther crypto.PublicKey) {
	tests := []struct {
		name        string
		signer      *tokenSigner
		verifier    *tokenVerifier
		verifyError string
	}{
		{
			name:     "sign",
			signer:   NewTokenSigner(alg, privKey, "4711").(*tokenSigner),
			verifier: NewTokenVerifier(pubKey).(*tokenVerifier),
		},
		{
			name:     "sign with audience",
			signer:   NewTokenSigner(alg, privKey, "4711", WithSignerAudience("www.example.com")).(*tokenSigner),
			verifier: NewTokenVerifier(pubKey, WithVerifierAudience("www.example.com")).(*tokenVerifier),
		},
		{
			name:     "sign with role",
			signer:   NewTokenSigner(alg, privKey, "4711", WithRole(RoleAgent)).(*tokenSigner),
			verifier: NewTokenVerifier(pubKey).(*tokenVerifier),
		},
		{
			name:     "sign with duration",
			signer:   NewTokenSigner(alg, privKey, "4711", WithTokenDuration(60*time.Second)).(*tokenSigner),
			verifier: NewTokenVerifier(pubKey).(*tokenVerifier),
		},
		{
			name:        "sign with missing audience",
			signer:      NewTokenSigner(alg, privKey, "4711").(*tokenSigner),
			verifier:    NewTokenVerifier(pubKey, WithVerifierAudience("www.example.com")).(*tokenVerifier),
			verifyError: "aud claim is required",
		},
		{
			name:        "verify invalid key",
			signer:      NewTokenSigner(alg, privKey, "4711").(*tokenSigner),
			verifier:    NewTokenVerifier(pubKeyOther).(*tokenVerifier),
			verifyError: "token signature is invalid",
		},
		{
			name:        "expired token",
			signer:      NewTokenSigner(alg, privKey, "4711", WithTokenDuration(-60*time.Second)).(*tokenSigner),
			verifier:    NewTokenVerifier(pubKey).(*tokenVerifier),
			verifyError: "token is expired",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokenString, err := tc.signer.SignToken()
			require.NoError(t, err)

			claims, err := tc.verifier.VerifyToken(tokenString)
			if tc.verifyError != "" {
				require.NotNil(t, err)
				require.Contains(t, err.Error(), tc.verifyError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, claims)
			require.Equal(t, claims.AgentID, tc.signer.agentID)
			require.Equal(t, claims.Role, string(tc.signer.role))
		})
	}
}
