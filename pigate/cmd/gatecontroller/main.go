package main

import (
	"context"
	"flag"
	"log"
	"time"

	"pigate/pkg/config"
	"pigate/pkg/database"
	"pigate/pkg/gate"
	"pigate/pkg/messenger"
)

const application string = "gatecontroller"

func main() {
	// 1) Parse command-line flags for config path
	var configFilePath string
	flag.StringVar(&configFilePath, "c", "/workspace/pigate/pkg/config",
		"Path to the configuration file")
	flag.Parse()

	// 2) Load configuration for gatecontroller
	cfg := config.LoadConfig(configFilePath, application+"-config").(config.GateControllerConfig)

	// 3) Initialize repository
	gm, err := database.NewSqliteGateManager(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database at %s: %v", cfg.DatabasePath, err)
	}
	defer gm.Close()

	// 4) Create GateController
	gateCtrl := gate.NewGateController(gm, cfg.GateOpenDuration)
	// Initialize the Raspberry Pi GPIO pin
	if err := gateCtrl.InitPinControl(cfg.RelayPin); err != nil {
		log.Fatalf("Failed to initialize GPIO pin: %v", err)
	}
	defer gateCtrl.Close()

	// 5) Start the keypad listener (non-blocking)
	keypadReader := gate.NewKeypadReader()
	go keypadReader.Start(func(code string) {
		if gateCtrl.ValidateCredential(code, time.Now()) {
			log.Printf("Valid credential: %s. Opening gate...", code)
			if err := gateCtrl.Open(); err != nil {
				log.Printf("Error opening gate: %v", err)
			}
		} else {
			log.Printf("Invalid credential: %s", code)
		}
	})

	// 6) Set up MQTT client
	client := messenger.NewMQTTClient(cfg.MQTTBroker, application, cfg.Location_ID)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect to MQTT broker (%s): %v", cfg.MQTTBroker, err)
	}
	defer client.Disconnect()

	// 7) Sync credentials on start
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.SyncCredentials(ctx, gm, cfg.Remote_DB_Table); err != nil {
		log.Println("Initial sync failed. Will retry later.")
	}

	// 8) Subscribe to updates via MQTT
	client.SubscribeCredentialStatus(database.HandleUpdateNotification(gm, cfg.Remote_DB_Table))
	client.SubscribePigateCommand(gateCtrl.CommandHandler())

	// Keep main go routine running (non-busy)
	select {}
}
