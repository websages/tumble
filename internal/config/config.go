package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Host            string        `yaml:"host" mapstructure:"host"`
	Database        string        `yaml:"database" mapstructure:"database"`
	Username        string        `yaml:"username" mapstructure:"username"`
	Password        string        `yaml:"password" mapstructure:"password"`
	BaseURL         string        `yaml:"baseurl" mapstructure:"baseurl"`
	Driver          string        `yaml:"driver" mapstructure:"driver"`
	Port            string        `yaml:"port" mapstructure:"port"`
	Mode            string        `yaml:"mode" mapstructure:"mode"`
	EmbedAssets     bool          `yaml:"embed_assets" mapstructure:"embed_assets"`
	ClickSigningKey string        `yaml:"click_signing_key" mapstructure:"click_signing_key"`
	AdminSecret     string        `yaml:"admin_secret" mapstructure:"admin_secret"`
	Logging         Logging       `yaml:"logging" mapstructure:"logging"`
	Caching         Caching       `yaml:"caching" mapstructure:"caching"`
	RequestTimeout  time.Duration `yaml:"request_timeout" mapstructure:"request_timeout"`
}

type Caching struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type Logging struct {
	Level  string `yaml:"level" mapstructure:"level"`   // debug, info, warn, error
	Output string `yaml:"output" mapstructure:"output"` // stdout, stderr, or file path
}

func Load(path string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("driver", "mysql")
	v.SetDefault("port", "8080")
	v.SetDefault("mode", "production")
	v.SetDefault("embed_assets", true)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("caching.enabled", true)
	v.SetDefault("request_timeout", "2s")

	// Environment Variables
	v.SetEnvPrefix("TUMBLE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Config File
	if path != "" {
		v.SetConfigFile(path)
	} else {
		// Search in common locations if no path provided
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("conf/")
		v.AddConfigPath("htdocs/")
		v.AddConfigPath(".")
	}

	// Read Config
	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, provided we have env vars or defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Normalize Mode
	if cfg.Mode == "dev" {
		cfg.Mode = "development"
	}

	return &cfg, nil
}

func (c *Config) DSN() string {
	if c.Driver == "sqlite" || c.Driver == "sqlite3" {
		// For SQLite, "Database" field is the file path
		// Add connection parameters for proper write access:
		// - journal_mode(WAL) enables Write-Ahead Logging for better concurrency
		// - busy_timeout(5000) waits up to 5 seconds if database is locked
		return c.Database + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	}
	// MySQL: username:password@tcp(host)/dbname?parseTime=true
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", c.Username, c.Password, c.Host, c.Database)
	return dsn
}
