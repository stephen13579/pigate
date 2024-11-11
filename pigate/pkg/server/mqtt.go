// mqtt.go

package server

import (
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type RealMQTTClient struct {
	client mqtt.Client
}

// Ensure RealMQTTClient satisfies the MQTTClientInterface
var _ MQTTClientInterface = (*RealMQTTClient)(nil)

func NewRealMQTTClient(broker string, clientID string) *RealMQTTClient {
	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID(clientID)
	client := mqtt.NewClient(opts)
	return &RealMQTTClient{client: client}
}

func (r *RealMQTTClient) Connect() error {
	token := r.client.Connect()
	token.Wait()
	return token.Error()
}

func (r *RealMQTTClient) Disconnect() {
	r.client.Disconnect(250)
}

func (r *RealMQTTClient) Publish(topic string, qos byte, retained bool, message interface{}) error {
	token := r.client.Publish(topic, qos, retained, message)
	token.Wait()
	return token.Error()
}

func (r *RealMQTTClient) Subscribe(topic string, qos byte, callback func(topic string, message interface{})) error {
	token := r.client.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
		callback(msg.Topic(), msg.Payload())
	})
	token.Wait()
	return token.Error()
}

func NotifyGateControllers(client MQTTClientInterface, topic string) error {
	err := client.Publish(topic, 1, true, "gate_credentials_update_available")
	if err != nil {
		log.Printf("Failed to notify gate controllers: %v", err)
		return err
	}
	log.Printf("Successfully notified gate controllers on topic: %s", topic)
	return nil
}
