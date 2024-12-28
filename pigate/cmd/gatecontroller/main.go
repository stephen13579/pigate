package main

import (
	"log"
	"time"

	"pigate/pkg/config"
	"pigate/pkg/database"
	"pigate/pkg/gate"
	"pigate/pkg/keypad"

	"pigate/pkg/updater"
)

func main() {
	// Load configurations
	cfg := config.LoadConfig(".")

	// Initialize the local database
	db, err := database.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize the gate controller
	gateCtrl, err := gate.NewGateController(cfg.GPIOPin)
	if err != nil {
		log.Fatalf("Failed to initialize gate controller: %v", err)
	}
	defer gateCtrl.Close()

	// Initialize the keypad reader
	keypadReader := keypad.NewKeypadReader()
	go keypadReader.Start(func(code string) {
		// Handle keypad input
		valid, err := database.ValidateCredential(db, code, time.Now()) // TODO what going on with tmie.now
		if err != nil {
			log.Printf("Error validating credential: %v", err)
			return
		}
		if valid {
			// Open the gate
			gateCtrl.Open(cfg.GateOpenDuration)
		} else {
			log.Printf("Invalid credential: %s", code)
		}
	})

	// Initialize MQTT client
	mqttClient := mqttclient.NewClient(cfg)
	err = mqttClient.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer mqttClient.Disconnect()

	// Subscribe to MQTT updates
	mqttClient.Subscribe(cfg.MQTTTopic, updater.HandleUpdateNotification)

	// Keep the application running
	select {}
}
