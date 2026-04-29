package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config contains application configuration loaded from environment variables.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Addr              string        `env:"SERVER_ADDR" env-default:":8080"`
	ReadTimeout       time.Duration `env:"SERVER_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout      time.Duration `env:"SERVER_WRITE_TIMEOUT" env-default:"10s"`
	ReadHeaderTimeout time.Duration `env:"SERVER_READ_HEADER_TIMEOUT" env-default:"3s"`
}

// DatabaseConfig contains PostgreSQL connection settings.
type DatabaseConfig struct {
	URL string `env:"DATABASE_URL" env-required:"true"`
}

// Load reads configuration from an optional config file and environment variables.
//
// If CONFIG_PATH is set, cleanenv reads variables from that file first.
// Environment variables from the current process override file values.
func Load() (Config, error) {
	var cfg Config

	configPath := os.Getenv("CONFIG_PATH")
	if configPath != "" {
		if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
			return Config{}, fmt.Errorf("read config file %q: %w", configPath, err)
		}
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("read environment variables: %w", err)
	}

	return cfg, nil
}
