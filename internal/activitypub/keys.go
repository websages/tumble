package activitypub

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"tumble/internal/data"
)

const keySize = 2048

// getOrCreateKey returns the site's ActivityPub signing keypair, generating
// and persisting one on first use.
func (s *Service) getOrCreateKey(ctx context.Context) (*data.ActivityPubKey, error) {
	key, err := s.Store.GetActivityPubKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("load activitypub key: %w", err)
	}
	if key != nil {
		return key, nil
	}

	privPEM, pubPEM, err := generateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate activitypub key: %w", err)
	}

	newKey := &data.ActivityPubKey{PrivateKey: privPEM, PublicKey: pubPEM}
	if err := s.Store.InsertActivityPubKey(ctx, newKey); err != nil {
		// Another process may have generated one concurrently; use theirs.
		if existing, gErr := s.Store.GetActivityPubKey(ctx); gErr == nil && existing != nil {
			return existing, nil
		}
		return nil, fmt.Errorf("save activitypub key: %w", err)
	}
	return newKey, nil
}

func generateKeyPair() (privPEM string, pubPEM string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return "", "", err
	}

	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}))

	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", err
	}
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}))

	return privPEM, pubPEM, nil
}

func parsePrivateKeyPEM(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block for private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func parsePublicKeyPEM(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block for public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}
	return rsaPub, nil
}
