package main

import (
	"log"

	"pigate/pkg/config"
	"pigate/pkg/server"
)

func main() {
	// Load configurations
	cfg := config.LoadConfig()

	// Start the HTTP server
	go func() {
		err := server.StartHTTPServer(cfg)
		if err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Start MQTT publisher
	mqttPub := server.NewMQTTPublisher(cfg)
	err := mqttPub.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer mqttPub.Disconnect()

	// Keep application running by causing goroutine to sleep
	select {}
}
