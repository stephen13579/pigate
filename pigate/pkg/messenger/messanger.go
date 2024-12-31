package messenger

import (
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
	client mqtt.Client
}

// Make sure the MQTTClient satisfies the MQTTClientInterface
var _ MQTTClientInterface = (*MQTTClient)(nil)

func NewMQTTClient(broker string, clientID string) *MQTTClient {
	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID(clientID)
	client := mqtt.NewClient(opts)
	return &MQTTClient{client: client}
}

func (r *MQTTClient) Connect() error {
	token := r.client.Connect()
	token.Wait()
	return token.Error()
}

func (r *MQTTClient) Disconnect() {
	r.client.Disconnect(250)
}

func (r *MQTTClient) Publish(topic string, qos byte, retained bool, message interface{}) error {
	token := r.client.Publish(topic, qos, retained, message)
	token.Wait()
	return token.Error()
}

func (r *MQTTClient) Subscribe(topic string, qos byte, callback func(topic string, msg mqtt.Message)) error {
	token := r.client.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
		callback(msg.Topic(), msg)
	})
	token.Wait()
	return token.Error()
}

func NotifyNewCredentials(client MQTTClientInterface, topic string) error {
	err := client.Publish(topic, 1, true, "gate_credentials_update_available")
	if err != nil {
		log.Printf("Failed to notify gate controllers: %v", err)
		return err
	}
	return nil
}

func CommandOpen(client MQTTClientInterface, topic string) error {
	err := client.Publish(topic, 1, true, "open")
	if err != nil {
		log.Printf("Failed to publish 'open' message: %v", err)
		return err
	}
	return nil
}

func CommandLockOpen(client MQTTClientInterface, topic string) error {
	err := client.Publish(topic, 1, true, "lock_open")
	if err != nil {
		log.Printf("Failed to publish 'lcok_open' message: %v", err)
		return err
	}
	return nil
}

func CommandClose(client MQTTClientInterface, topic string) error {
	err := client.Publish(topic, 1, true, "close")
	if err != nil {
		log.Printf("Failed to publish 'close' message: %v", err)
		return err
	}
	return nil
}
