package mqttingest

import (
	"context"
	"encoding/json"
	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/ingestion"
	"fmt"
	"log/slog"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// TelemetrySink accepts telemetry messages for further processing.
type TelemetrySink interface {
	Submit(ctx context.Context, input ingestion.TelemetryInput) error
}

// Subscriber receives telemetry readings from MQTT broker.
type Subscriber struct {
	logger *slog.Logger
	cfg    config.MQTTConfig
	sink   TelemetrySink
}

// NewSubscriber creates MQTT telemetry subscriber.
func NewSubscriber(
	logger *slog.Logger,
	cfg config.MQTTConfig,
	sink TelemetrySink,
) *Subscriber {
	return &Subscriber{
		logger: logger,
		cfg:    cfg,
		sink:   sink,
	}
}

// Start connects to MQTT broker and subscribes to telemetry topic.
func (s *Subscriber) Start(ctx context.Context) error {
	options := paho.NewClientOptions().
		AddBroker(s.cfg.BrokerURL).
		SetClientID(s.cfg.ClientID).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectTimeout(s.cfg.ConnectTimeout)

	client := paho.NewClient(options)

	connectToken := client.Connect()
	if !connectToken.WaitTimeout(s.cfg.ConnectTimeout) {
		return fmt.Errorf("mqtt connect timeout after %s", s.cfg.ConnectTimeout)
	}

	if err := connectToken.Error(); err != nil {
		return fmt.Errorf("mqtt connect: %w", err)
	}

	s.logger.Info(
		"connected to mqtt broker",
		"brokerUrl", s.cfg.BrokerURL,
		"topic", s.cfg.TelemetryTopic,
		"clientId", s.cfg.ClientID,
	)

	subscribeToken := client.Subscribe(
		s.cfg.TelemetryTopic,
		byte(s.cfg.QoS),
		func(_ paho.Client, message paho.Message) {
			if err := s.handlePayload(context.Background(), message.Payload()); err != nil {
				s.logger.Error(
					"enqueue mqtt telemetry message failed",
					"topic", message.Topic(),
					"error", err,
				)
			}
		},
	)

	if !subscribeToken.WaitTimeout(5 * time.Second) {
		client.Disconnect(250)
		return fmt.Errorf("mqtt subscribe timeout")
	}

	if err := subscribeToken.Error(); err != nil {
		client.Disconnect(250)
		return fmt.Errorf("mqtt subscribe: %w", err)
	}

	s.logger.Info("subscribed to mqtt topic", "topic", s.cfg.TelemetryTopic)

	<-ctx.Done()

	client.Unsubscribe(s.cfg.TelemetryTopic).WaitTimeout(2 * time.Second)
	client.Disconnect(250)

	return nil
}

func (s *Subscriber) handlePayload(ctx context.Context, payload []byte) error {
	var input ingestion.TelemetryInput

	if err := json.Unmarshal(payload, &input); err != nil {
		return fmt.Errorf("decode mqtt telemetry payload: %w", err)
	}

	if err := s.sink.Submit(ctx, input); err != nil {
		return fmt.Errorf("submit mqtt telemetry payload: %w", err)
	}

	return nil
}
