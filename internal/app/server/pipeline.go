package server

import (
	"context"
	"errors"
	"log/slog"

	kafkaadapter "extrusion-quality-system/internal/adapters/kafka"
	mqttadapter "extrusion-quality-system/internal/adapters/mqtt"
	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/usecase/telemetry"
)

func startTelemetryPipeline(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
	telemetryService *telemetry.Service,
) error {
	var kafkaProducer *kafkaadapter.Producer

	if cfg.Kafka.Enabled {
		kafkaProducer = kafkaadapter.NewProducer(logger, cfg.Kafka)

		kafkaConsumer := kafkaadapter.NewConsumer(
			logger,
			cfg.Kafka,
			telemetryService,
		)

		go func() {
			<-ctx.Done()

			if err := kafkaProducer.Close(); err != nil {
				logger.Error("close kafka producer failed", "error", err)
			}

			if err := kafkaConsumer.Close(); err != nil {
				logger.Error("close kafka consumer failed", "error", err)
			}
		}()

		go func() {
			if err := kafkaConsumer.Start(ctx); err != nil {
				logger.Error("kafka consumer stopped with error", "error", err)
			}
		}()

		logger.Info(
			"kafka pipeline enabled",
			"brokers", cfg.Kafka.BrokerList(),
			"topic", cfg.Kafka.TelemetryTopic,
			"consumerGroup", cfg.Kafka.ConsumerGroup,
		)
	}

	logger.Info(
		"mqtt config loaded",
		"enabled", cfg.MQTT.Enabled,
		"brokerUrl", cfg.MQTT.BrokerURL,
		"topic", cfg.MQTT.TelemetryTopic,
		"qos", cfg.MQTT.QoS,
	)

	if cfg.MQTT.Enabled {
		if !cfg.Kafka.Enabled {
			return errors.New("mqtt pipeline requires kafka to be enabled")
		}

		if kafkaProducer == nil {
			return errors.New("kafka producer is not configured")
		}

		mqttSubscriber := mqttadapter.NewSubscriber(
			logger,
			cfg.MQTT,
			kafkaProducer,
		)

		go func() {
			if err := mqttSubscriber.Start(ctx); err != nil {
				logger.Error("mqtt subscriber stopped with error", "error", err)
			}
		}()

		logger.Info(
			"mqtt to kafka bridge enabled",
			"mqttBrokerUrl", cfg.MQTT.BrokerURL,
			"mqttTopic", cfg.MQTT.TelemetryTopic,
			"kafkaTopic", cfg.Kafka.TelemetryTopic,
		)
	}

	return nil
}
