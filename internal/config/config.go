package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config contains application configuration loaded from environment variables.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	MQTT     MQTTConfig
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

// MQTTConfig contains MQTT subscriber settings for backend telemetry ingestion.
type MQTTConfig struct {
	Enabled        bool          `env:"MQTT_ENABLED" env-default:"false"`
	BrokerURL      string        `env:"MQTT_BROKER_URL" env-default:"tcp://localhost:1883"`
	ClientID       string        `env:"MQTT_CLIENT_ID" env-default:"extrusion-backend"`
	TelemetryTopic string        `env:"MQTT_TELEMETRY_TOPIC" env-default:"extrusion/telemetry/readings"`
	QoS            int           `env:"MQTT_QOS" env-default:"1"`
	ConnectTimeout time.Duration `env:"MQTT_CONNECT_TIMEOUT" env-default:"5s"`
}

// Load reads configuration from .env and environment variables.
// Real environment variables have priority over values from .env.
func Load() (Config, error) {
	var cfg Config

	configPath := resolveConfigPath()

	if err := loadDotEnv(configPath); err != nil {
		return Config{}, fmt.Errorf("load env file %q: %w", configPath, err)
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("read environment variables: %w", err)
	}

	return cfg, nil
}
