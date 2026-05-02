package mqttadapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"extrusion-quality-system/internal/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type TelemetryPublisher interface {
	PublishTelemetry(ctx context.Context, key string, payload []byte) error
}

type Subscriber struct {
	logger    *slog.Logger
	cfg       config.MQTTConfig
	publisher TelemetryPublisher
}

func NewSubscriber(
	logger *slog.Logger,
	cfg config.MQTTConfig,
	publisher TelemetryPublisher,
) *Subscriber {
	return &Subscriber{
		logger:    logger,
		cfg:       cfg,
		publisher: publisher,
	}
}

func (s *Subscriber) Start(ctx context.Context) error {
	if s.publisher == nil {
		return fmt.Errorf("mqtt telemetry publisher is not configured")
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(s.cfg.BrokerURL)
	opts.SetClientID(s.cfg.ClientID)
	opts.SetConnectTimeout(s.cfg.ConnectTimeout)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(2 * time.Second)

	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		s.logger.Error("mqtt connection lost", "error", err)
	})

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		s.logger.Info(
			"mqtt connected",
			"brokerUrl", s.cfg.BrokerURL,
			"topic", s.cfg.TelemetryTopic,
		)

		token := client.Subscribe(
			s.cfg.TelemetryTopic,
			s.cfg.QoS,
			s.handleMessage,
		)

		if !token.WaitTimeout(s.cfg.ConnectTimeout) {
			s.logger.Error(
				"mqtt subscribe timeout",
				"topic", s.cfg.TelemetryTopic,
				"timeout", s.cfg.ConnectTimeout,
			)
			return
		}

		if err := token.Error(); err != nil {
			s.logger.Error(
				"mqtt subscribe failed",
				"topic", s.cfg.TelemetryTopic,
				"error", err,
			)
			return
		}

		s.logger.Info(
			"mqtt subscribed",
			"topic", s.cfg.TelemetryTopic,
			"qos", s.cfg.QoS,
		)
	})

	client := mqtt.NewClient(opts)

	token := client.Connect()
	if !token.WaitTimeout(s.cfg.ConnectTimeout) {
		return fmt.Errorf("mqtt connect timeout after %s", s.cfg.ConnectTimeout)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("connect to mqtt broker: %w", err)
	}

	<-ctx.Done()

	client.Disconnect(250)

	return nil
}

func (s *Subscriber) handleMessage(_ mqtt.Client, message mqtt.Message) {
	payload := append([]byte(nil), message.Payload()...)

	messageCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ConnectTimeout)
	defer cancel()

	if err := s.publisher.PublishTelemetry(messageCtx, message.Topic(), payload); err != nil {
		s.logger.Error(
			"publish mqtt telemetry to kafka failed",
			"mqttTopic", message.Topic(),
			"payloadSize", len(payload),
			"error", err,
		)
		return
	}

	s.logger.Debug(
		"mqtt telemetry forwarded to kafka",
		"mqttTopic", message.Topic(),
		"payloadSize", len(payload),
	)
}
