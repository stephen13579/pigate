// mqtt_test.go

package server_test

import (
	"pigate/pkg/config"
	"pigate/pkg/server"
	"pigate/pkg/testutils"
	"testing"
)

func TestNotifyGateControllers(t *testing.T) {
	cfg := &config.Config{
		MQTTTopic: "gate/notifications",
	}

	mockClient := testutils.NewMockMQTTClient()
	mockClient.Connect()

	err := server.NotifyGateControllers(mockClient, cfg.MQTTTopic)
	if err != nil {
		t.Errorf("Expected NotifyGateControllers to succeed, got error: %v", err)
	}
	if len(mockClient.Messages) != 1 {
		t.Errorf("Expected one published message, got %d", len(mockClient.Messages))
	}
	// Subscribe to get message
	received := ""
	mockClient.Subscribe(cfg.MQTTTopic, 1, func(topic string, message interface{}) {
		received = message.(string)
	})
	if received != "gate_credentials_update_available" {
		t.Errorf("Expected message content to match, got %s", received)
	}
}

func TestMQTTSubscribe(t *testing.T) {
	mockClient := testutils.NewMockMQTTClient()
	mockClient.Connect()

	received := ""

	mockClient.Subscribe("test/topic", 1, func(topic string, message interface{}) {
		received = message.(string)
	})

	// Publish message to test/topic
	mockClient.Publish("test/topic", 1, false, "hello world!")

	if received != "hello world!" {
		t.Errorf("Expected 'hello world', got '%s'", received)
	}
}
