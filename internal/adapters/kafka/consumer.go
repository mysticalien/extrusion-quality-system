package kafkaadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/ingestion"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	logger           *slog.Logger
	reader           *kafka.Reader
	ingestionService *ingestion.Service
	retryDelay       time.Duration
	topic            string
	groupID          string
}

func NewConsumer(
	logger *slog.Logger,
	cfg config.KafkaConfig,
	ingestionService *ingestion.Service,
) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.BrokerList(),
		Topic:    cfg.TelemetryTopic,
		GroupID:  cfg.ConsumerGroup,
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  cfg.ReadTimeout,
	})

	return &Consumer{
		logger:           logger,
		reader:           reader,
		ingestionService: ingestionService,
		retryDelay:       cfg.RetryDelay,
		topic:            cfg.TelemetryTopic,
		groupID:          cfg.ConsumerGroup,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info(
		"kafka consumer started",
		"topic", c.topic,
		"groupId", c.groupID,
	)

	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}

			c.logger.Error(
				"fetch kafka message failed",
				"topic", c.topic,
				"groupId", c.groupID,
				"error", err,
			)

			time.Sleep(c.retryDelay)
			continue
		}

		if err := c.handleMessage(ctx, message); err != nil {
			c.logger.Error(
				"process kafka telemetry message failed",
				"topic", message.Topic,
				"partition", message.Partition,
				"offset", message.Offset,
				"error", err,
			)

			if ingestion.IsValidationError(err) {
				if commitErr := c.reader.CommitMessages(ctx, message); commitErr != nil {
					c.logger.Error(
						"commit invalid kafka message failed",
						"topic", message.Topic,
						"partition", message.Partition,
						"offset", message.Offset,
						"error", commitErr,
					)
				}
			}

			time.Sleep(c.retryDelay)
			continue
		}

		if err := c.reader.CommitMessages(ctx, message); err != nil {
			c.logger.Error(
				"commit kafka message failed",
				"topic", message.Topic,
				"partition", message.Partition,
				"offset", message.Offset,
				"error", err,
			)

			time.Sleep(c.retryDelay)
			continue
		}

		c.logger.Debug(
			"kafka telemetry message committed",
			"topic", message.Topic,
			"partition", message.Partition,
			"offset", message.Offset,
		)
	}
}

func (c *Consumer) handleMessage(ctx context.Context, message kafka.Message) error {
	var input ingestion.TelemetryInput

	if err := json.Unmarshal(message.Value, &input); err != nil {
		if commitErr := c.reader.CommitMessages(ctx, message); commitErr != nil {
			c.logger.Error(
				"commit malformed kafka message failed",
				"topic", message.Topic,
				"partition", message.Partition,
				"offset", message.Offset,
				"error", commitErr,
			)
		}

		return fmt.Errorf("decode telemetry input from kafka message: %w", err)
	}

	c.logger.Debug(
		"kafka telemetry message received",
		"topic", message.Topic,
		"partition", message.Partition,
		"offset", message.Offset,
		"parameterType", input.ParameterType,
		"sourceId", input.SourceID,
		"measuredAt", input.MeasuredAt,
	)

	_, err := c.ingestionService.Process(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

func (c *Consumer) Close() error {
	if c.reader == nil {
		return nil
	}

	return c.reader.Close()
}
