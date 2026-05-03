package simulator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"extrusion-quality-system/internal/config"
)

type mqttSender struct {
	client paho.Client
	topic  string
	qos    byte
}

func newMQTTSender(cfg config.SimulatorConfig) (*mqttSender, error) {
	options := paho.NewClientOptions().
		AddBroker(cfg.MQTTBrokerURL).
		SetClientID(cfg.MQTTClientID).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectTimeout(cfg.RequestTimeout)

	client := paho.NewClient(options)

	token := client.Connect()
	if !token.WaitTimeout(cfg.RequestTimeout) {
		return nil, fmt.Errorf("mqtt connect timeout after %s", cfg.RequestTimeout)
	}

	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect: %w", err)
	}

	return &mqttSender{
		client: client,
		topic:  cfg.MQTTTopic,
		qos:    byte(cfg.MQTTQoS),
	}, nil
}

func (s *mqttSender) Send(_ context.Context, reading telemetryMessage) error {
	body, err := json.Marshal(reading)
	if err != nil {
		return fmt.Errorf("marshal telemetry reading: %w", err)
	}

	token := s.client.Publish(s.topic, s.qos, false, body)
	if !token.WaitTimeout(5 * time.Second) {
		return errors.New("mqtt publish timeout")
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt publish: %w", err)
	}

	return nil
}

func (s *mqttSender) Close() {
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(250)
	}
}
