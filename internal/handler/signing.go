package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateClickSignature creates an HMAC signature for a link ID.
// The signature is used to verify that a click came from a legitimately
// rendered page rather than being artificially generated.
func GenerateClickSignature(id int, secret string) string {
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d", id)))
	// Use first 16 hex chars (8 bytes) - sufficient for integrity, keeps URLs short
	return hex.EncodeToString(mac.Sum(nil))[:16]
}

// ValidateClickSignature checks if the provided signature matches
// the expected signature for the given ID and secret.
// Returns true if the signature is valid, false otherwise.
func ValidateClickSignature(id int, signature, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}
	expected := GenerateClickSignature(id, secret)
	// Use constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(signature), []byte(expected))
}
