package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Host     string  `yaml:"host"`
	Database string  `yaml:"database"`
	Username string  `yaml:"username"`
	Password string  `yaml:"password"`
	BaseURL  string  `yaml:"baseurl"`
	Driver   string  `yaml:"driver"`
	Logging  Logging `yaml:"logging"`
}

type Logging struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Output string `yaml:"output"` // stdout, stderr, or file path
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	// Defaults
	if cfg.Driver == "" {
		cfg.Driver = "mysql"
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
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
