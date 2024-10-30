package main

import (
	"log"

	"pigate/pkg/config"
	"pigate/pkg/server"
)

func main() {
	// Load configurations
	cfg := config.LoadConfig()

	// Start the HTTP server to receive new credential files
	go func() {
		err := server.StartHTTPServer(cfg)
		if err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Initialize MQTT publisher
	mqttPub := server.NewMQTTPublisher(cfg)
	err := mqttPub.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer mqttPub.Disconnect()

	// Keep the application running
	select {}
}
