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

// TestLoad_EnvOnly verifies that the app can be configured entirely from
// environment variables with no config file present. This is required for
// container deployments (e.g. Fly.io) that provide config via env.
// Keys without a SetDefault must still be populated from TUMBLE_* vars.
func TestLoad_EnvOnly(t *testing.T) {
	env := map[string]string{
		"TUMBLE_DRIVER":            "sqlite",
		"TUMBLE_DATABASE":          "/data/tumble.sqlite",
		"TUMBLE_BASEURL":           "https://tumble2.wcyd.org",
		"TUMBLE_ADMIN_SECRET":      "s3cret",
		"TUMBLE_CLICK_SIGNING_KEY": "signing",
		"TUMBLE_HOST":              "db.internal",
		"TUMBLE_USERNAME":          "tumble",
		"TUMBLE_PASSWORD":          "pw",
		"TUMBLE_SITE_TITLE":        "Tumble",
	}
	for k, v := range env {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	checks := map[string]struct{ got, want string }{
		"Database":        {cfg.Database, "/data/tumble.sqlite"},
		"BaseURL":         {cfg.BaseURL, "https://tumble2.wcyd.org"},
		"AdminSecret":     {cfg.AdminSecret, "s3cret"},
		"ClickSigningKey": {cfg.ClickSigningKey, "signing"},
		"Host":            {cfg.Host, "db.internal"},
		"Username":        {cfg.Username, "tumble"},
		"Password":        {cfg.Password, "pw"},
		"SiteTitle":       {cfg.SiteTitle, "Tumble"},
	}
	for name, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: expected %q, got %q", name, c.want, c.got)
		}
	}
}
