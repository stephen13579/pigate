// mqtt_client_mock.go

package testutils

import (
	"errors"
	"pigate/pkg/server"
)

type MockMQTTClient struct {
	Connected bool
	Messages  map[string]string // [topic]message (only if retain == true)
	callbacks map[string]func(topic string, message interface{})
}

func NewMockMQTTClient() *MockMQTTClient {
	return &MockMQTTClient{
		Connected: false,
		Messages:  make(map[string]string),
		callbacks: make(map[string]func(topic string, message interface{})),
	}
}

// Ensure MockMQTTClient satisfies the MQTTClientInterface
var _ server.MQTTClientInterface = (*MockMQTTClient)(nil)

func (m *MockMQTTClient) Connect() error {
	m.Connected = true
	return nil
}

func (m *MockMQTTClient) Disconnect() {
	m.Connected = false
}

func (m *MockMQTTClient) Publish(topic string, qos byte, retained bool, message interface{}) error {
	if !m.Connected {
		return errors.New("not connected to broker")
	}
	// Store the message if retain is true
	if retained {
		m.Messages[topic] = message.(string)
	}
	// Trigger callbacks for the specific topic, if any exist
	if callback, exists := m.callbacks[topic]; exists {
		callback(topic, message)
	}
	return nil
}

func (m *MockMQTTClient) Subscribe(topic string, qos byte, callback func(topic string, message interface{})) error {
	if !m.Connected {
		return errors.New("not connected to broker")
	}
	// Store the callback function for the topic
	m.callbacks[topic] = callback
	// Check if there are any pending messages for this topic
	if message, exists := m.Messages[topic]; exists {
		callback(topic, message)
	}
	return nil
}

// Helper method to simulate receiving a message on a subscribed topic
func (m *MockMQTTClient) SimulateMessageReceive(topic string, message interface{}) {
	if callback, exists := m.callbacks[topic]; exists {
		callback(topic, message)
	}
}
