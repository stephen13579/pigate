package server

import (
	"pigate/pkg/config"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type MQTTPublisher struct {
	client MQTT.Client
}

func NewMQTTPublisher(cfg *config.Config) *MQTTPublisher {
	opts := MQTT.NewClientOptions()
	opts.AddBroker(cfg.MQTTBroker)
	opts.SetClientID("pigate_publisher")
	return &MQTTPublisher{
		client: MQTT.NewClient(opts),
	}
}

func (p *MQTTPublisher) Connect() error {
	if token := p.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (p *MQTTPublisher) Disconnect() {
	p.client.Disconnect(250)
}

func NotifyGateControllers() error {
	cfg := config.LoadConfig()
	publisher := NewMQTTPublisher(cfg)
	err := publisher.Connect()
	if err != nil {
		return err
	}
	defer publisher.Disconnect()

	token := publisher.client.Publish(cfg.MQTTTopic, 1, false, "update_available")
	token.Wait()
	return token.Error()
}
