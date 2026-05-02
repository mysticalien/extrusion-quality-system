package kafkaingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"extrusion-quality-system/internal/config"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	logger *slog.Logger
	writer *kafka.Writer
	topic  string
}

func NewProducer(logger *slog.Logger, cfg config.KafkaConfig) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.BrokerList()...),
		Topic:        cfg.TelemetryTopic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: cfg.WriteTimeout,
		ReadTimeout:  cfg.ReadTimeout,
	}

	return &Producer{
		logger: logger,
		writer: writer,
		topic:  cfg.TelemetryTopic,
	}
}

func (p *Producer) PublishTelemetry(
	ctx context.Context,
	key string,
	payload []byte,
) error {
	if len(payload) == 0 {
		return fmt.Errorf("kafka telemetry payload is empty")
	}

	err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("write telemetry message to kafka: %w", err)
	}

	p.logger.Debug(
		"telemetry message published to kafka",
		"topic", p.topic,
		"key", key,
		"payloadSize", len(payload),
	)

	return nil
}

func (p *Producer) Close() error {
	if p.writer == nil {
		return nil
	}

	return p.writer.Close()
}
