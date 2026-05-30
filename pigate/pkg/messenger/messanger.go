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
	return NewMQTTClientWithCredentials(broker, clientID, locationID, "", "")
}

func NewMQTTClientWithCredentials(broker string, clientID string, locationID string, username string, password string) *MQTTClient {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetCleanSession(false).
		SetConnectRetry(true).
		SetConnectRetryInterval(30 * time.Second)

	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}

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
	return waitForToken("connect to MQTT broker", token)
}

func (r *MQTTClient) Disconnect() {
	r.client.Disconnect(250)
}

func (r *MQTTClient) IsConnected() bool {
	return r.client.IsConnectionOpen()
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
	StatusOpened     = "opened"
	StatusLockedOpen = "locked_open"
	StatusClosed     = "closed"
)

// Status messages (payloads) for `locationID/credentials/status`
const (
	UpdateAvailable = "update_available"
)

func (r *MQTTClient) NotifyNewCredentials() error {
	topic := fmt.Sprintf(TopicCredentialsStatus, r.locationID)
	if err := r.publish(topic, true, UpdateAvailable); err != nil {
		log.Printf("Failed to notify gate controllers: %v", err)
		return err
	}
	return nil
}

func (r *MQTTClient) CommandOpen() error {
	topic := fmt.Sprintf(TopicPigateCommand, r.locationID)
	if err := r.publish(topic, false, CommandOpenMessage); err != nil {
		log.Printf("Failed to publish 'open' message: %v", err)
		return err
	}
	return nil
}

func (r *MQTTClient) CommandLockOpen() error {
	topic := fmt.Sprintf(TopicPigateCommand, r.locationID)
	if err := r.publish(topic, false, CommandHoldOpenMessage); err != nil {
		log.Printf("Failed to publish 'lock_open' message: %v", err)
		return err
	}
	return nil
}

func (r *MQTTClient) CommandClose() error {
	topic := fmt.Sprintf(TopicPigateCommand, r.locationID)
	if err := r.publish(topic, false, CommandCloseMessage); err != nil {
		log.Printf("Failed to publish 'close' message: %v", err)
		return err
	}
	return nil
}

func (r *MQTTClient) NotifyGateOpen() error {
	topic := fmt.Sprintf(TopicPigateStatus, r.locationID)
	if err := r.publish(topic, true, StatusOpened); err != nil {
		log.Printf("Failed to publish 'opened' status: %v", err)
		return err
	}
	return nil
}

func (r *MQTTClient) NotifyGateLockedOpen() error {
	topic := fmt.Sprintf(TopicPigateStatus, r.locationID)
	if err := r.publish(topic, true, StatusLockedOpen); err != nil {
		log.Printf("Failed to publish 'locked_open' status: %v", err)
		return err
	}
	return nil
}

func (r *MQTTClient) NotifyGateClosed() error {
	topic := fmt.Sprintf(TopicPigateStatus, r.locationID)
	if err := r.publish(topic, true, StatusClosed); err != nil {
		log.Printf("Failed to publish 'closed' status: %v", err)
		return err
	}
	return nil
}

func (r *MQTTClient) publish(topic string, retained bool, payload string) error {
	token := r.client.Publish(topic, 1, retained, payload)
	return waitForToken(fmt.Sprintf("publish to topic %s", topic), token)
}

func waitForToken(action string, token mqtt.Token) error {
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("%s timed out", action)
	}
	return token.Error()
}

func (r *MQTTClient) SubscribePigateStatus(callback func(topic string, message string)) error {
	topic := fmt.Sprintf(TopicPigateStatus, r.locationID)

	r.mu.Lock()
	r.subscriptions[topic] = func(client mqtt.Client, msg mqtt.Message) {
		callback(msg.Topic(), string(msg.Payload()))
	}
	r.mu.Unlock()

	token := r.client.Subscribe(topic, 1, r.subscriptions[topic])

	if err := waitForToken(fmt.Sprintf("subscribe to topic %s", topic), token); err != nil {
		log.Printf("Failed to subscribe to topic '%s': %v", topic, err)
		return err
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

	if err := waitForToken(fmt.Sprintf("subscribe to topic %s", topic), token); err != nil {
		log.Printf("Failed to subscribe to topic '%s': %v", topic, err)
		return err
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

	if err := waitForToken(fmt.Sprintf("subscribe to topic %s", topic), token); err != nil {
		log.Printf("Failed to subscribe to topic '%s': %v", topic, err)
		return err
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
		if err := waitForToken(fmt.Sprintf("resubscribe to topic %s", topic), token); err != nil {
			log.Printf("Failed to resubscribe to topic '%s': %v", topic, err)
		} else {
			log.Printf("Resubscribed to '%s'", topic)
		}
	}
}
