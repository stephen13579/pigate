package main

import (
	"context"
	"flag"
	"fmt"
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
	cfg := config.LoadConfig(configFilePath, application+"-config").(*config.GateControllerConfig)

	// print config
	log.Printf("Loaded configuration: %+v", cfg)

	// 3) Initialize repository
	gm, err := database.NewSqliteGateManager(cfg.LocalDBPath)
	if err != nil {
		log.Fatalf("Failed to open database at %s: %v", cfg.LocalDBPath, err)
	}
	defer gm.Close()

	// 4) Create GateController
	gateCtrl := gate.NewGateController(gm, cfg.GateOpenDuration)
	// Initialize the Raspberry Pi GPIO pin
	ledPinNumber := 27
	gateCtrl.InitPinControl(cfg.RelayPin, ledPinNumber)
	defer gateCtrl.Close()

	// 5) Start the keypad listener (non-blocking)
	keypadReader := gate.NewKeypadReader()
	err = keypadReader.Start(func(code string) {
		if gateCtrl.ValidateCredential(code, time.Now()) {
			log.Printf("Credential %s validated successfully", code)
		} else {
			log.Printf("Failed to validate credential %s", code)
		}
	})
	if err != nil {
		log.Fatalf("Failed to start keypad reader: %v", err)
	}

	// 6) Set up MQTT client
	client := messenger.NewMQTTClient(cfg.MQTTBroker, application, cfg.Location_ID)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect to MQTT broker (%s): %v", cfg.MQTTBroker, err)
	}
	defer client.Disconnect()

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name)

	// 7) Sync credentials on start
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.SyncCredentials(ctx, gm, connStr); err != nil {
		log.Println("Initial sync failed. Will retry later.")
	}

	// 8) Periodic sync every 24 hours
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			<-ticker.C
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := database.SyncCredentials(ctx, gm, connStr); err != nil {
				log.Println("Periodic sync failed:", err)
			} else {
				log.Println("Periodic sync completed successfully.")
			}
			cancel()
		}
	}()

	// 9) Subscribe to updates via MQTT
	client.SubscribeCredentialStatus(database.HandleUpdateNotification(gm, connStr))
	client.SubscribePigateCommand(gateCtrl.CommandHandler())

	// Keep main go routine running (non-busy)
	select {}
}
