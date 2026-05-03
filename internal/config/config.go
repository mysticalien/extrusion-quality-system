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
	Logging   LoggingConfig
	Kafka     KafkaConfig
}

type KafkaConfig struct {
	Enabled        bool          `env:"KAFKA_ENABLED" env-default:"false"`
	Brokers        string        `env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	TelemetryTopic string        `env:"KAFKA_TELEMETRY_TOPIC" env-default:"extrusion.telemetry.raw"`
	ConsumerGroup  string        `env:"KAFKA_CONSUMER_GROUP" env-default:"extrusion-quality-service"`
	WriteTimeout   time.Duration `env:"KAFKA_WRITE_TIMEOUT" env-default:"10s"`
	ReadTimeout    time.Duration `env:"KAFKA_READ_TIMEOUT" env-default:"10s"`
	RetryDelay     time.Duration `env:"KAFKA_RETRY_DELAY" env-default:"2s"`
}

type LoggingConfig struct {
	Level string `env:"LOG_LEVEL" env-default:"info"`
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
	TokenSecret string        `env:"JWT_SECRET" env-default:"local-development-secret-change-me-32-chars-min"`
	TokenTTL    time.Duration `env:"JWT_TOKEN_TTL" env-default:"24h"`
	TokenIssuer string        `env:"AUTH_TOKEN_ISSUER" env-default:"extrusion-quality-system"`
	BcryptCost  int           `env:"AUTH_BCRYPT_COST" env-default:"10"`
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
	Mode           string        `env:"SIMULATOR_MODE" env-default:"normal"`
	Period         time.Duration `env:"SIMULATOR_INTERVAL" env-default:"2s"`
	RequestTimeout time.Duration `env:"SIMULATOR_REQUEST_TIMEOUT" env-default:"10s"`

	SourceID string `env:"SIMULATOR_SOURCE_ID" env-default:"http-simulator"`

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

	if cfg.Auth.TokenIssuer == "" {
		return fmt.Errorf("AUTH_TOKEN_ISSUER must not be empty")
	}

	if cfg.Auth.BcryptCost < 4 || cfg.Auth.BcryptCost > 16 {
		return fmt.Errorf("AUTH_BCRYPT_COST must be between 4 and 16")
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

	if cfg.Simulator.MQTTQoS > 2 {
		return fmt.Errorf("SIMULATOR_MQTT_QOS must be 0, 1 or 2")
	}

	if cfg.Simulator.MQTTTimeout <= 0 {
		return fmt.Errorf("SIMULATOR_MQTT_TIMEOUT must be positive")
	}

	if cfg.Kafka.Enabled {
		if len(cfg.Kafka.BrokerList()) == 0 {
			return fmt.Errorf("KAFKA_BROKERS must not be empty when Kafka is enabled")
		}

		if cfg.Kafka.TelemetryTopic == "" {
			return fmt.Errorf("KAFKA_TELEMETRY_TOPIC must not be empty when Kafka is enabled")
		}

		if cfg.Kafka.ConsumerGroup == "" {
			return fmt.Errorf("KAFKA_CONSUMER_GROUP must not be empty when Kafka is enabled")
		}

		if cfg.Kafka.WriteTimeout <= 0 {
			return fmt.Errorf("KAFKA_WRITE_TIMEOUT must be positive")
		}

		if cfg.Kafka.ReadTimeout <= 0 {
			return fmt.Errorf("KAFKA_READ_TIMEOUT must be positive")
		}

		if cfg.Kafka.RetryDelay <= 0 {
			return fmt.Errorf("KAFKA_RETRY_DELAY must be positive")
		}
	}

	switch strings.ToLower(cfg.Logging.Level) {
	case "debug", "info", "warn", "error":
		// ok
	default:
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn or error")
	}

	return nil
}

func (c KafkaConfig) BrokerList() []string {
	rawBrokers := strings.Split(c.Brokers, ",")
	brokers := make([]string, 0, len(rawBrokers))

	for _, broker := range rawBrokers {
		broker = strings.TrimSpace(broker)
		if broker == "" {
			continue
		}

		brokers = append(brokers, broker)
	}

	return brokers
}
