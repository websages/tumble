package activitypub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/doyensec/safeurl"
	"github.com/go-fed/httpsig"
)

var signedHeaders = []string{httpsig.RequestTarget, "host", "date", "digest"}

// signRequest attaches an HTTP Signature (RFC draft used by ActivityPub/
// Mastodon) and a Digest header to an outgoing request, signed with the
// site's private key.
func (s *Service) signRequest(ctx context.Context, req *http.Request, body []byte) error {
	key, err := s.getOrCreateKey(ctx)
	if err != nil {
		return err
	}
	priv, err := parsePrivateKeyPEM(key.PrivateKey)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, signedHeaders, httpsig.Signature, 0)
	if err != nil {
		return err
	}

	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	if req.Host == "" {
		req.Host = req.URL.Host
	}
	// go-fed/httpsig reads the "Host" value straight from the header map
	// when signing (unlike NewVerifier, which special-cases r.Host).
	req.Header.Set("Host", req.Host)

	return signer.SignRequest(priv, s.publicKeyID(), req, body)
}

// newSafeClient returns an SSRF-safe HTTP client for outbound federation
// requests (actor lookups and inbox deliveries).
func newSafeClient(timeout time.Duration) *safeurl.WrappedClient {
	cfg := safeurl.GetConfigBuilder().SetTimeout(timeout).Build()
	return safeurl.Client(cfg)
}

// fetchRemoteActor retrieves and parses a remote actor document.
func (s *Service) fetchRemoteActor(ctx context.Context, actorURL string) (*Actor, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, actorURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json")

	client := newSafeClient(10 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch actor %s: status %d", actorURL, resp.StatusCode)
	}

	var actor Actor
	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		return nil, fmt.Errorf("decode actor %s: %w", actorURL, err)
	}
	return &actor, nil
}

// verifyInboundSignature verifies the HTTP Signature on an inbound inbox
// request, fetching the sender's public key from their actor document.
// It returns the verified remote actor.
func (s *Service) verifyInboundSignature(r *http.Request) (*Actor, error) {
	verifier, err := httpsig.NewVerifier(r)
	if err != nil {
		return nil, fmt.Errorf("parse signature: %w", err)
	}

	keyID := verifier.KeyId()
	actorURL := keyID
	if idx := strings.Index(keyID, "#"); idx != -1 {
		actorURL = keyID[:idx]
	}

	actor, err := s.fetchRemoteActor(r.Context(), actorURL)
	if err != nil {
		return nil, fmt.Errorf("fetch signer actor: %w", err)
	}
	if actor.PublicKey.PublicKeyPem == "" {
		return nil, fmt.Errorf("actor %s has no public key", actorURL)
	}

	pubKey, err := parsePublicKeyPEM(actor.PublicKey.PublicKeyPem)
	if err != nil {
		return nil, fmt.Errorf("parse signer public key: %w", err)
	}

	if err := verifier.Verify(pubKey, httpsig.RSA_SHA256); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return actor, nil
}
