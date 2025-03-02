package messenger

import (
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
	client        mqtt.Client
	locationID    string
	subscriptions map[string]mqtt.MessageHandler // store callbacks for re-connecting after network loss
	mu            sync.Mutex                     // For accessing subscriptions map
}

// Make sure the MQTTClient satisfies the MQTTClientInterface
var _ MQTTClientInterface = (*MQTTClient)(nil)

func NewMQTTClient(broker string, clientID string, locationID string) *MQTTClient {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetCleanSession(false).
		SetConnectRetry(true).
		SetConnectRetryInterval(30 * time.Second)

	r := &MQTTClient{
		client:        mqtt.NewClient(opts),
		locationID:    locationID,
		subscriptions: make(map[string]mqtt.MessageHandler),
	}

	// Handle successful connection
	opts.OnConnect = func(c mqtt.Client) {
		log.Println("MQTT Connected!")
		r.resubscribeAll() // 🔹 Restore previous subscriptions
	}

	// Handle lost connection
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT Connection lost: %v. Retrying...", err)
	}

	return r
}

func (r *MQTTClient) Connect() error {
	token := r.client.Connect()
	token.Wait()
	return token.Error()
}

func (r *MQTTClient) Disconnect() {
	r.client.Disconnect(250)
}

// Topic templates
const (
	TopicPigateStatus      = "%s/pigate/status"      // e.g. "location123/pigate/status"
	TopicPigateCommand     = "%s/pigate/command"     // e.g. "location123/pigate/command"
	TopicCredentialsStatus = "%s/credentials/status" // e.g. "location123/credentials/status"
)

// Command messages (payloads) for `locationID/pigate/command`
const (
	CommandOpenMessage     = "open"
	CommandHoldOpenMessage = "hold_open"
	CommandCloseMessage    = "close"
)

// Status messages (payloads) for `locationID/pigate/status`
const (
	StatusOpened = "opened"
	StatusClosed = "closed"
)

// Status messages (payloads) for `locationID/credentials/status`
const (
	UpdateAvailable = "update_available"
)

func (r *MQTTClient) NotifyNewCredentials() error {
	topic := fmt.Sprintf(TopicCredentialsStatus, r.locationID)
	err := r.client.Publish(topic, 1, true, UpdateAvailable)
	if err.Error() != nil {
		log.Printf("Failed to notify gate controllers: %v", err.Error())
		return err.Error()
	}
	return nil
}

func (r *MQTTClient) CommandOpen() error {
	topic := fmt.Sprintf(TopicPigateCommand, r.locationID)
	err := r.client.Publish(topic, 1, true, CommandOpenMessage)
	if err.Error() != nil {
		log.Printf("Failed to publish 'open' message: %v", err.Error())
		return err.Error()
	}
	return nil
}

func (r *MQTTClient) CommandLockOpen() error {
	topic := fmt.Sprintf(TopicPigateCommand, r.locationID)
	err := r.client.Publish(topic, 1, true, CommandHoldOpenMessage)
	if err.Error() != nil {
		log.Printf("Failed to publish 'lcok_open' message: %v", err.Error())
		return err.Error()
	}
	return nil
}

func (r *MQTTClient) CommandClose() error {
	topic := fmt.Sprintf(TopicPigateCommand, r.locationID)
	err := r.client.Publish(topic, 1, true, CommandCloseMessage)
	if err != nil {
		log.Printf("Failed to publish 'close' message: %v", err.Error())
		return err.Error()
	}
	return nil
}

func (r *MQTTClient) NotifyGateOpen() error {
	topic := fmt.Sprintf(TopicPigateStatus, r.locationID)
	err := r.client.Publish(topic, 1, true, StatusOpened)
	if err.Error() != nil {
		log.Printf("Failed to publish 'close' message: %v", err.Error())
		return err.Error()
	}
	return nil
}

func (r *MQTTClient) NotifyGateClosed() error {
	topic := fmt.Sprintf(TopicPigateStatus, r.locationID)
	err := r.client.Publish(topic, 1, true, StatusClosed)
	if err.Error() != nil {
		log.Printf("Failed to publish 'close' message: %v", err.Error())
		return err.Error()
	}
	return nil
}

func (r *MQTTClient) SubscribePigateStatus(callback func(topic string, message string)) error {
	topic := fmt.Sprintf(TopicPigateStatus, r.locationID)

	r.mu.Lock()
	r.subscriptions[topic] = func(client mqtt.Client, msg mqtt.Message) {
		callback(msg.Topic(), string(msg.Payload()))
	}
	r.mu.Unlock()

	token := r.client.Subscribe(topic, 1, r.subscriptions[topic])
	token.Wait()

	if token.Error() != nil {
		log.Printf("Failed to subscribe to topic '%s': %v", topic, token.Error())
		return token.Error()
	}

	log.Printf("Subscribed to '%s' for pigate status updates", topic)
	return nil
}

func (r *MQTTClient) SubscribePigateCommand(callback func(topic string, command string)) error {
	topic := fmt.Sprintf(TopicPigateCommand, r.locationID)

	r.mu.Lock()
	r.subscriptions[topic] = func(client mqtt.Client, msg mqtt.Message) {
		callback(msg.Topic(), string(msg.Payload()))
	}
	r.mu.Unlock()

	token := r.client.Subscribe(topic, 1, r.subscriptions[topic])
	token.Wait()

	if token.Error() != nil {
		log.Printf("Failed to subscribe to topic '%s': %v", topic, token.Error())
		return token.Error()
	}

	log.Printf("Subscribed to '%s' for pigate commands", topic)
	return nil
}

func (r *MQTTClient) SubscribeCredentialStatus(callback func(topic string, status string)) error {
	topic := fmt.Sprintf(TopicCredentialsStatus, r.locationID)

	r.mu.Lock()
	r.subscriptions[topic] = func(client mqtt.Client, msg mqtt.Message) {
		callback(msg.Topic(), string(msg.Payload()))
	}
	r.mu.Unlock()

	token := r.client.Subscribe(topic, 1, r.subscriptions[topic])
	token.Wait()

	if token.Error() != nil {
		log.Printf("Failed to subscribe to topic '%s': %v", topic, token.Error())
		return token.Error()
	}

	log.Printf("Subscribed to '%s' for credential status updates", topic)
	return nil
}

func (r *MQTTClient) resubscribeAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	log.Println("Resubscribing to all MQTT topics...")

	for topic, callback := range r.subscriptions {
		token := r.client.Subscribe(topic, 1, callback)
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to resubscribe to topic '%s': %v", topic, token.Error())
		} else {
			log.Printf("Resubscribed to '%s'", topic)
		}
	}
}
