package config

import (
	"os"
	"testing"
)

func TestLoad_DevAlias(t *testing.T) {
	// Set environment variable for test
	os.Setenv("TUMBLE_MODE", "dev")
	defer os.Unsetenv("TUMBLE_MODE")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Mode != "development" {
		t.Errorf("Expected Mode to be 'development', got '%s'", cfg.Mode)
	}
}
