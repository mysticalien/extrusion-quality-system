package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// SimulatorConfig contains telemetry simulator settings.
type SimulatorConfig struct {
	Transport      string        `env:"SIMULATOR_TRANSPORT" env-default:"http"`
	BackendURL     string        `env:"SIMULATOR_BACKEND_URL" env-default:"http://localhost:8080"`
	MQTTBrokerURL  string        `env:"SIMULATOR_MQTT_BROKER_URL" env-default:"tcp://localhost:1883"`
	MQTTClientID   string        `env:"SIMULATOR_MQTT_CLIENT_ID" env-default:"extrusion-simulator"`
	MQTTTopic      string        `env:"SIMULATOR_MQTT_TOPIC" env-default:"extrusion/telemetry/readings"`
	MQTTQoS        int           `env:"SIMULATOR_MQTT_QOS" env-default:"1"`
	Mode           string        `env:"SIMULATOR_MODE" env-default:"normal"`
	Period         time.Duration `env:"SIMULATOR_PERIOD" env-default:"2s"`
	SourceID       string        `env:"SIMULATOR_SOURCE_ID" env-default:"simulator"`
	RequestTimeout time.Duration `env:"SIMULATOR_REQUEST_TIMEOUT" env-default:"5s"`
}

// LoadSimulator reads simulator configuration from .env and environment variables.
// Real environment variables have priority over values from .env.
func LoadSimulator() (SimulatorConfig, error) {
	var cfg SimulatorConfig

	configPath := resolveConfigPath()

	if err := loadDotEnv(configPath); err != nil {
		return SimulatorConfig{}, fmt.Errorf("load env file %q: %w", configPath, err)
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return SimulatorConfig{}, fmt.Errorf("read simulator environment variables: %w", err)
	}

	return cfg, nil
}
