package messenger_test

import (
	"testing"
	"time"

	"pigate/pkg/config"
	"pigate/pkg/messenger"
)

func TestNotifyNewCredentials(t *testing.T) {
	cfg := &config.CredentialServerConfig{
		MQTTBroker:  "tcp://emqx:1883",
		Location_ID: "test-location-id",
	}

	clientID := "test-client"
	client := messenger.NewMQTTClient(cfg.MQTTBroker, clientID, cfg.Location_ID)

	// Connect to MQTT
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Subscribe to credential status updates
	var received string
	err := client.SubscribeCredentialStatus(func(topic, status string) {
		received = status
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to credential status updates: %v", err)
	}

	// Publish the "update_available" message
	if err := client.NotifyNewCredentials(); err != nil {
		t.Fatalf("Failed to notify new credentials: %v", err)
	}

	// Wait briefly to allow the message to arrive
	time.Sleep(200 * time.Millisecond)

	// Verify the received message
	if received != messenger.UpdateAvailable {
		t.Errorf("Expected payload '%s', got '%s'", messenger.UpdateAvailable, received)
	}
}

func TestCommandOpen(t *testing.T) {
	cfg := &config.CredentialServerConfig{
		MQTTBroker:  "tcp://emqx:1883",
		Location_ID: "test-location-id",
	}

	clientID := "test-client"
	client := messenger.NewMQTTClient(cfg.MQTTBroker, clientID, cfg.Location_ID)

	// Connect to MQTT
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Subscribe to pigate command
	var received string
	err := client.SubscribePigateCommand(func(topic, command string) {
		received = command
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pigate command: %v", err)
	}

	// Publish the "open" command
	if err := client.CommandOpen(); err != nil {
		t.Fatalf("Failed to command open: %v", err)
	}

	// Wait briefly to allow the message to arrive
	time.Sleep(200 * time.Millisecond)

	// Verify the received message
	if received != messenger.CommandOpenMessage {
		t.Errorf("Expected payload '%s', got '%s'", messenger.CommandOpenMessage, received)
	}
}

func TestCommandLockOpen(t *testing.T) {
	cfg := &config.CredentialServerConfig{
		MQTTBroker:  "tcp://emqx:1883",
		Location_ID: "test-location-id",
	}

	clientID := "test-client"
	client := messenger.NewMQTTClient(cfg.MQTTBroker, clientID, cfg.Location_ID)

	// Connect to MQTT
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Subscribe to pigate command
	var received string
	err := client.SubscribePigateCommand(func(topic, command string) {
		received = command
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pigate command: %v", err)
	}

	// Publish the "hold_open" command
	if err := client.CommandLockOpen(); err != nil {
		t.Fatalf("Failed to command lock open: %v", err)
	}

	// Wait briefly to allow the message to arrive
	time.Sleep(200 * time.Millisecond)

	// Verify the received message
	if received != messenger.CommandHoldOpenMessage {
		t.Errorf("Expected payload '%s', got '%s'", messenger.CommandHoldOpenMessage, received)
	}
}

func TestCommandClose(t *testing.T) {
	cfg := &config.CredentialServerConfig{
		MQTTBroker:  "tcp://emqx:1883",
		Location_ID: "test-location-id",
	}

	clientID := "test-client"
	client := messenger.NewMQTTClient(cfg.MQTTBroker, clientID, cfg.Location_ID)

	// Connect to MQTT
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Subscribe to pigate command
	var received string
	err := client.SubscribePigateCommand(func(topic, command string) {
		received = command
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pigate command: %v", err)
	}

	// Publish the "close" command
	if err := client.CommandClose(); err != nil {
		t.Fatalf("Failed to command close: %v", err)
	}

	// Wait briefly to allow the message to arrive
	time.Sleep(200 * time.Millisecond)

	// Verify the received message
	if received != messenger.CommandCloseMessage {
		t.Errorf("Expected payload '%s', got '%s'", messenger.CommandCloseMessage, received)
	}
}

func TestNotifyGateOpen(t *testing.T) {
	cfg := &config.CredentialServerConfig{
		MQTTBroker:  "tcp://emqx:1883",
		Location_ID: "test-location-id",
	}

	clientID := "test-client"
	client := messenger.NewMQTTClient(cfg.MQTTBroker, clientID, cfg.Location_ID)

	// Connect to MQTT
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Subscribe to pigate status
	var received string
	err := client.SubscribePigateStatus(func(topic, message string) {
		received = message
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pigate status: %v", err)
	}

	// Publish the "opened" status
	if err := client.NotifyGateOpen(); err != nil {
		t.Fatalf("Failed to notify gate open: %v", err)
	}

	// Wait briefly to allow the message to arrive
	time.Sleep(200 * time.Millisecond)

	// Verify the received message
	if received != messenger.StatusOpened {
		t.Errorf("Expected payload '%s', got '%s'", messenger.StatusOpened, received)
	}
}

func TestNotifyGateClosed(t *testing.T) {
	cfg := &config.CredentialServerConfig{
		MQTTBroker:  "tcp://emqx:1883",
		Location_ID: "test-location-id",
	}

	clientID := "test-client"
	client := messenger.NewMQTTClient(cfg.MQTTBroker, clientID, cfg.Location_ID)

	// Connect to MQTT
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Subscribe to pigate status
	var received string
	err := client.SubscribePigateStatus(func(topic, message string) {
		received = message
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pigate status: %v", err)
	}

	// Publish the "closed" status
	if err := client.NotifyGateClosed(); err != nil {
		t.Fatalf("Failed to notify gate closed: %v", err)
	}

	// Wait briefly to allow the message to arrive
	time.Sleep(200 * time.Millisecond)

	// Verify the received message
	if received != messenger.StatusClosed {
		t.Errorf("Expected payload '%s', got '%s'", messenger.StatusClosed, received)
	}
}

func TestSubscribePigateCommand(t *testing.T) {
	cfg := &config.CredentialServerConfig{
		MQTTBroker:  "tcp://emqx:1883",
		Location_ID: "test-location-id",
	}

	clientID := "test-client"
	client := messenger.NewMQTTClient(cfg.MQTTBroker, clientID, cfg.Location_ID)

	// Connect to MQTT
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Subscribe to pigate status
	var received string
	err := client.SubscribePigateCommand(func(topic, message string) {
		received = message
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pigate status: %v", err)
	}

	// Publish a test message
	if err := client.CommandOpen(); err != nil {
		t.Fatalf("Failed to publish message to MQTT broker: %v", err)
	}

	// Wait briefly to allow the message to arrive
	time.Sleep(200 * time.Millisecond)

	// Verify the received message
	if received != messenger.CommandOpenMessage {
		t.Errorf("Expected payload '%s', got '%s'", messenger.StatusOpened, received)
	}
}

func TestSubscribePigateStatus(t *testing.T) {
	cfg := &config.CredentialServerConfig{
		MQTTBroker:  "tcp://emqx:1883",
		Location_ID: "test-location-id",
	}

	clientID := "test-client"
	client := messenger.NewMQTTClient(cfg.MQTTBroker, clientID, cfg.Location_ID)

	// Connect to MQTT
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Subscribe to pigate status
	var received string
	err := client.SubscribePigateStatus(func(topic, message string) {
		received = message
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pigate status: %v", err)
	}

	// Publish a test message
	if err := client.NotifyGateOpen(); err != nil {
		t.Fatalf("Failed to publish message to MQTT broker: %v", err)
	}

	// Wait briefly to allow the message to arrive
	time.Sleep(200 * time.Millisecond)

	// Verify the received message
	if received != messenger.StatusOpened {
		t.Errorf("Expected payload '%s', got '%s'", messenger.StatusOpened, received)
	}
}

func TestSubscribeCredentialStatus(t *testing.T) {
	cfg := &config.CredentialServerConfig{
		MQTTBroker:  "tcp://emqx:1883",
		Location_ID: "test-location-id",
	}

	clientID := "test-client"
	client := messenger.NewMQTTClient(cfg.MQTTBroker, clientID, cfg.Location_ID)

	// Connect to MQTT
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Subscribe to pigate status
	var received string
	err := client.SubscribeCredentialStatus(func(topic, message string) {
		received = message
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pigate status: %v", err)
	}

	// Publish a test message
	if err := client.NotifyNewCredentials(); err != nil {
		t.Fatalf("Failed to publish message to MQTT broker: %v", err)
	}

	// Wait briefly to allow the message to arrive
	time.Sleep(200 * time.Millisecond)

	// Verify the received message
	if received != messenger.UpdateAvailable {
		t.Errorf("Expected payload '%s', got '%s'", messenger.StatusOpened, received)
	}
}
