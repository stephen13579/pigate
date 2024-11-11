// mqtt_interface.go

package server

type MQTTClientInterface interface {
	Connect() error
	Disconnect()
	Publish(topic string, qos byte, retained bool, message interface{}) error
	Subscribe(topic string, qos byte, callback func(topic string, message interface{})) error
}
