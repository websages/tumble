package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Host     string  `yaml:"host" mapstructure:"host"`
	Database string  `yaml:"database" mapstructure:"database"`
	Username string  `yaml:"username" mapstructure:"username"`
	Password string  `yaml:"password" mapstructure:"password"`
	BaseURL  string  `yaml:"baseurl" mapstructure:"baseurl"`
	Driver   string  `yaml:"driver" mapstructure:"driver"`
	Port     string  `yaml:"port" mapstructure:"port"`
	Mode     string  `yaml:"mode" mapstructure:"mode"`
	Logging  Logging `yaml:"logging" mapstructure:"logging"`
	Caching  Caching `yaml:"caching" mapstructure:"caching"`
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
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("caching.enabled", true)

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
		return c.Database
	}
	// MySQL: username:password@tcp(host)/dbname?parseTime=true
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", c.Username, c.Password, c.Host, c.Database)
	return dsn
}
