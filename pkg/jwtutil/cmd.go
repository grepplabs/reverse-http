package jwtutil

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"io"
	"os"

	"github.com/grepplabs/cert-source/tls/keyutil"
	"github.com/grepplabs/reverse-http/config"
)

func GeneratePrivateKey(conf *config.AuthKeyPrivateCmd) error {
	_, privateKeyPEM, _, _, err := generateKeys(TokenAlg(conf.Algo))
	if err != nil {
		return err
	}
	return writeToFile(conf.OutputFile, privateKeyPEM)
}

func GeneratePublicKey(conf *config.AuthKeyPublicCmd) error {
	privateKey, err := keyutil.ReadPrivateKeyFile(conf.InputFile)
	if err != nil {
		return err
	}
	key, ok := privateKey.(interface {
		Public() crypto.PublicKey
	})
	if !ok {
		return fmt.Errorf("invalid private key type: %T", key)
	}
	publicKeyPEM, err := keyutil.MarshalPublicKeyToPEM(key.Public())
	if err != nil {
		return err
	}
	return writeToFile(conf.OutputFile, publicKeyPEM)
}
func GenerateJWTToken(conf *config.AuthJwtTokenCmd) error {
	privateKey, err := keyutil.ReadPrivateKeyFile(conf.InputFile)
	if err != nil {
		return err
	}
	alg, err := tokenAlgFromPrivateKey(privateKey)
	if err != nil {
		return err
	}
	signer := NewTokenSigner(alg, privateKey, conf.AgentID,
		WithSignerAudience(conf.Audience),
		WithTokenDuration(conf.Duration),
		WithRole(Role(conf.Role)))

	tokenString, err := signer.SignToken()
	if err != nil {
		return err
	}
	return writeToFile(conf.OutputFile, []byte(tokenString))
}

func writeToFile(outputFile string, content []byte) error {
	var output io.Writer
	if outputFile == "-" {
		output = os.Stdout
	} else {
		f, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				_, _ = fmt.Fprint(os.Stderr, err.Error())
			}
		}()
		output = f
	}
	_, err := io.Copy(output, bytes.NewReader(content))
	if err != nil {
		return err
	}
	return nil
}

func generateKeys(alg TokenAlg) (crypto.PrivateKey, []byte, crypto.PublicKey, []byte, error) {
	switch alg {
	case ES256:
		return keyutil.GenerateECKeys()
	case RS256:
		return keyutil.GenerateRSAKeys()
	default:
		return nil, nil, nil, nil, fmt.Errorf("unsupported alg: %s", alg)
	}
}

func tokenAlgFromPrivateKey(privateKey crypto.PrivateKey) (TokenAlg, error) {
	switch privateKey.(type) {
	case *ecdsa.PrivateKey:
		return ES256, nil
	case *rsa.PrivateKey:
		return RS256, nil
	default:
		return "", fmt.Errorf("private key is not a recognized type: %T", privateKey)
	}
}
