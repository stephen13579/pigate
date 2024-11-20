// mqtt_interface.go

package mqttclient

import mqtt "github.com/eclipse/paho.mqtt.golang"

type MQTTClientInterface interface {
	Connect() error
	Disconnect()
	Publish(topic string, qos byte, retained bool, message interface{}) error
	Subscribe(topic string, qos byte, callback func(topic string, msg mqtt.Message)) error
}
