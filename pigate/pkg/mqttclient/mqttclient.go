package mqttclient

import (
	"pigate/pkg/config"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	client MQTT.Client
}

func NewClient(cfg *config.Config) *Client {
	opts := MQTT.NewClientOptions()
	opts.AddBroker(cfg.MQTTBroker)
	opts.SetClientID("pigate_client")
	return &Client{
		client: MQTT.NewClient(opts),
	}
}

func (c *Client) Connect() error {
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c *Client) Disconnect() {
	c.client.Disconnect(250)
}

func (c *Client) Subscribe(topic string, handler MQTT.MessageHandler) error {
	if token := c.client.Subscribe(topic, 1, handler); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
