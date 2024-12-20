// mqtt_test.go

package mqttclient

import (
	"pigate/pkg/config"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestNotifyGateController(t *testing.T) {
	cfg := &config.Config{
		MQTTTopic:  "test/update",
		MQTTBroker: "tcp://emqx:1883",
	}

	clientID := "test-client"
	client := NewMQTTClient(cfg.MQTTBroker, clientID)

	// Publish before connection
	err := NotifyNewCredentials(client, cfg.MQTTTopic)
	if err == nil {
		t.Errorf("Expected NotifyGateControllers to fail, but err == nil")
	}

	err = client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	var received string
	err = client.Subscribe(cfg.MQTTTopic, 1, func(topic string, msg mqtt.Message) {
		received = string(msg.Payload())
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Publish the message
	err = NotifyNewCredentials(client, cfg.MQTTTopic)
	if err != nil {
		t.Errorf("Expected NotifyGateControllers to succeed, got error: %v", err)
	}

	// Pause to allow the message to be received
	time.Sleep(100 * time.Millisecond)

	if received != "gate_credentials_update_available" {
		t.Errorf("Expected message content to match, got '%s'", received)
	}
}

func TestCommandOpen(t *testing.T) {
	cfg := &config.Config{
		MQTTTopic:  "test/command",
		MQTTBroker: "tcp://emqx:1883",
	}

	clientID := "test-client"
	client := NewMQTTClient(cfg.MQTTBroker, clientID)

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	var received string
	err = client.Subscribe(cfg.MQTTTopic, 1, func(topic string, msg mqtt.Message) {
		received = string(msg.Payload())
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	err = CommandOpen(client, cfg.MQTTTopic)
	if err != nil {
		t.Fatalf("Failed to command open: %v", err)
	}

	// Pause to allow the message to be received
	time.Sleep(100 * time.Millisecond)

	if received != "open" {
		t.Errorf("Expected message content to match, got '%s'", received)
	}
}

func TestCommandLockOpen(t *testing.T) {
	cfg := &config.Config{
		MQTTTopic:  "test/command",
		MQTTBroker: "tcp://emqx:1883",
	}

	clientID := "test-client"
	client := NewMQTTClient(cfg.MQTTBroker, clientID)

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	var received string
	err = client.Subscribe(cfg.MQTTTopic, 1, func(topic string, msg mqtt.Message) {
		received = string(msg.Payload())
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	err = CommandLockOpen(client, cfg.MQTTTopic)
	if err != nil {
		t.Fatalf("Failed to command lock_open: %v", err)
	}

	// Pause to allow the message to be received
	time.Sleep(100 * time.Millisecond)

	if received != "lock_open" {
		t.Errorf("Expected message content to match, got '%s'", received)
	}
}

func TestCommandClose(t *testing.T) {
	cfg := &config.Config{
		MQTTTopic:  "test/command",
		MQTTBroker: "tcp://emqx:1883",
	}

	clientID := "test-client"
	client := NewMQTTClient(cfg.MQTTBroker, clientID)

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	var received string
	err = client.Subscribe(cfg.MQTTTopic, 1, func(topic string, msg mqtt.Message) {
		received = string(msg.Payload())
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	err = CommandClose(client, cfg.MQTTTopic)
	if err != nil {
		t.Fatalf("Failed to command close: %v", err)
	}

	// Pause to allow the message to be received
	time.Sleep(100 * time.Millisecond)

	if received != "close" {
		t.Errorf("Expected message content to match, got '%s'", received)
	}
}
