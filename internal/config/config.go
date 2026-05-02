package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Auth      AuthConfig
	MQTT      MQTTConfig
	Simulator SimulatorConfig
}

type ServerConfig struct {
	Addr              string        `env:"SERVER_ADDR" env-default:":8080"`
	ReadTimeout       time.Duration `env:"SERVER_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout      time.Duration `env:"SERVER_WRITE_TIMEOUT" env-default:"10s"`
	ReadHeaderTimeout time.Duration `env:"SERVER_READ_HEADER_TIMEOUT" env-default:"5s"`
}

type DatabaseConfig struct {
	URL string `env:"DATABASE_URL" env-default:"postgres://postgres:postgres@localhost:5432/extrusion_quality?sslmode=disable"`
}

type AuthConfig struct {
	TokenSecret string        `env:"JWT_SECRET" env-default:"local-dev-secret-change-me"`
	TokenTTL    time.Duration `env:"JWT_TOKEN_TTL" env-default:"24h"`
}

type MQTTConfig struct {
	Enabled        bool          `env:"MQTT_ENABLED" env-default:"false"`
	BrokerURL      string        `env:"MQTT_BROKER_URL" env-default:"tcp://localhost:1883"`
	TelemetryTopic string        `env:"MQTT_TELEMETRY_TOPIC" env-default:"extrusion/telemetry"`
	ClientID       string        `env:"MQTT_CLIENT_ID" env-default:"extrusion-quality-server"`
	WorkerCount    int           `env:"MQTT_WORKER_COUNT" env-default:"4"`
	QueueSize      int           `env:"MQTT_QUEUE_SIZE" env-default:"100"`
	ConnectTimeout time.Duration `env:"MQTT_CONNECT_TIMEOUT" env-default:"10s"`
	QoS            byte          `env:"MQTT_QOS" env-default:"0"`
}

type SimulatorConfig struct {
	Transport      string        `env:"SIMULATOR_TRANSPORT" env-default:"http"`
	Mode           string        `env:"SIMULATOR_MODE" env-default:"http"`
	Period         time.Duration `env:"SIMULATOR_INTERVAL" env-default:"2s"`
	RequestTimeout time.Duration `env:"SIMULATOR_REQUEST_TIMEOUT" env-default:"10s"`

	BackendURL string `env:"SIMULATOR_TARGET_URL" env-default:"http://localhost:8080/api/telemetry"`
	SourceID   string `env:"SIMULATOR_SOURCE_ID" env-default:"http-simulator"`
	AuthToken  string `env:"SIMULATOR_AUTH_TOKEN" env-default:""`

	MQTTBrokerURL string        `env:"SIMULATOR_MQTT_BROKER_URL" env-default:"tcp://localhost:1883"`
	MQTTClientID  string        `env:"SIMULATOR_MQTT_CLIENT_ID" env-default:"extrusion-simulator"`
	MQTTTopic     string        `env:"SIMULATOR_MQTT_TOPIC" env-default:"extrusion/telemetry"`
	MQTTQoS       byte          `env:"SIMULATOR_MQTT_QOS" env-default:"0"`
	MQTTTimeout   time.Duration `env:"SIMULATOR_MQTT_TIMEOUT" env-default:"10s"`
}

func Load() (Config, error) {
	var cfg Config

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = ".env"
	}

	if _, err := os.Stat(configPath); err == nil {
		if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
			return Config{}, fmt.Errorf("read config file %s: %w", configPath, err)
		}

		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return Config{}, fmt.Errorf("read environment config: %w", err)
		}
	} else if os.IsNotExist(err) {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return Config{}, fmt.Errorf("read environment config: %w", err)
		}
	} else {
		return Config{}, fmt.Errorf("check config file %s: %w", configPath, err)
	}

	applySimulatorEnvAliases(&cfg)

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func LoadSimulator() (SimulatorConfig, error) {
	cfg, err := Load()
	if err != nil {
		return SimulatorConfig{}, err
	}

	return cfg.Simulator, nil
}

func applySimulatorEnvAliases(cfg *Config) {
	if value := strings.TrimSpace(os.Getenv("SIMULATOR_PERIOD")); value != "" {
		period, err := time.ParseDuration(value)
		if err == nil {
			cfg.Simulator.Period = period
		}
	}

	if value := strings.TrimSpace(os.Getenv("SIMULATOR_BACKEND_URL")); value != "" {
		cfg.Simulator.BackendURL = value
	}

	if value := strings.TrimSpace(os.Getenv("SIMULATOR_MQTT_BROKER")); value != "" {
		cfg.Simulator.MQTTBrokerURL = value
	}
}

func validate(cfg Config) error {
	if cfg.Server.Addr == "" {
		return fmt.Errorf("SERVER_ADDR must not be empty")
	}

	if cfg.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL must not be empty")
	}

	if cfg.Auth.TokenSecret == "" {
		return fmt.Errorf("JWT_SECRET must not be empty")
	}

	if cfg.Auth.TokenTTL <= 0 {
		return fmt.Errorf("JWT_TOKEN_TTL must be positive")
	}

	if cfg.MQTT.WorkerCount <= 0 {
		return fmt.Errorf("MQTT_WORKER_COUNT must be positive")
	}

	if cfg.MQTT.QueueSize <= 0 {
		return fmt.Errorf("MQTT_QUEUE_SIZE must be positive")
	}

	if cfg.MQTT.ConnectTimeout <= 0 {
		return fmt.Errorf("MQTT_CONNECT_TIMEOUT must be positive")
	}

	if cfg.MQTT.QoS > 2 {
		return fmt.Errorf("MQTT_QOS must be 0, 1 or 2")
	}

	if cfg.Simulator.Period <= 0 {
		return fmt.Errorf("SIMULATOR_INTERVAL must be positive")
	}

	if cfg.Simulator.RequestTimeout <= 0 {
		return fmt.Errorf("SIMULATOR_REQUEST_TIMEOUT must be positive")
	}

	if cfg.Simulator.Mode != "http" && cfg.Simulator.Mode != "mqtt" {
		return fmt.Errorf("SIMULATOR_MODE must be http or mqtt")
	}

	if cfg.Simulator.MQTTQoS > 2 {
		return fmt.Errorf("SIMULATOR_MQTT_QOS must be 0, 1 or 2")
	}

	if cfg.Simulator.MQTTTimeout <= 0 {
		return fmt.Errorf("SIMULATOR_MQTT_TIMEOUT must be positive")
	}

	return nil
}
