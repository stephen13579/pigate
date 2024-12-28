package updater

import (
	"log"
	"net/http"

	"pigate/pkg/config"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	//"pigate/pkg/database"
)

func HandleUpdateNotification(client MQTT.Client, msg MQTT.Message) {
	log.Println("Received update notification via MQTT")
	err := FetchAndUpdateCredentials()
	if err != nil {
		log.Printf("Error updating credentials: %v", err)
	}
}

func FetchAndUpdateCredentials() error {
	// Fetch the updated credentials from the server
	resp, err := http.Get(config.LoadConfig(".").ServerURL + "/credentials")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Update the local database
	// For now, just log the action
	log.Println("Fetched updated credentials from server")
	return nil
}
