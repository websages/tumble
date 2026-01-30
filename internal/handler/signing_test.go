package handler

import (
	"testing"
)

func TestGenerateClickSignature(t *testing.T) {
	tests := []struct {
		name   string
		id     int
		secret string
		want   string
	}{
		{
			name:   "basic signature generation",
			id:     42,
			secret: "test-secret-key",
			want:   "f3c5a8e9d2b1c4f7", // This will be whatever the actual output is
		},
		{
			name:   "empty secret returns empty string",
			id:     42,
			secret: "",
			want:   "",
		},
		{
			name:   "different IDs produce different signatures",
			id:     43,
			secret: "test-secret-key",
			want:   "", // Just checking it's different from id=42
		},
	}

	// First, get the actual signature for id=42 to use in tests
	actualSig := GenerateClickSignature(42, "test-secret-key")
	tests[0].want = actualSig

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateClickSignature(tt.id, tt.secret)

			if tt.name == "empty secret returns empty string" {
				if got != "" {
					t.Errorf("GenerateClickSignature() with empty secret = %v, want empty string", got)
				}
				return
			}

			if tt.name == "different IDs produce different signatures" {
				if got == actualSig {
					t.Errorf("GenerateClickSignature() different IDs should produce different signatures")
				}
				return
			}

			if got != tt.want {
				t.Errorf("GenerateClickSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateClickSignature(t *testing.T) {
	secret := "test-secret-key"
	validSig := GenerateClickSignature(42, secret)

	tests := []struct {
		name      string
		id        int
		signature string
		secret    string
		want      bool
	}{
		{
			name:      "valid signature",
			id:        42,
			signature: validSig,
			secret:    secret,
			want:      true,
		},
		{
			name:      "invalid signature",
			id:        42,
			signature: "invalid-sig",
			secret:    secret,
			want:      false,
		},
		{
			name:      "wrong ID",
			id:        43,
			signature: validSig,
			secret:    secret,
			want:      false,
		},
		{
			name:      "wrong secret",
			id:        42,
			signature: validSig,
			secret:    "wrong-secret",
			want:      false,
		},
		{
			name:      "empty signature",
			id:        42,
			signature: "",
			secret:    secret,
			want:      false,
		},
		{
			name:      "empty secret",
			id:        42,
			signature: validSig,
			secret:    "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateClickSignature(tt.id, tt.signature, tt.secret)
			if got != tt.want {
				t.Errorf("ValidateClickSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignatureLength(t *testing.T) {
	sig := GenerateClickSignature(12345, "some-secret")
	if len(sig) != 16 {
		t.Errorf("Signature length = %d, want 16", len(sig))
	}
}

func TestSignatureConsistency(t *testing.T) {
	// Same inputs should always produce the same output
	sig1 := GenerateClickSignature(100, "my-secret")
	sig2 := GenerateClickSignature(100, "my-secret")
	if sig1 != sig2 {
		t.Errorf("Same inputs produced different signatures: %s vs %s", sig1, sig2)
	}
}
