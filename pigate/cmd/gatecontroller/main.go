package main

import (
	"context"
	"flag"
	"log"
	"time"

	"pigate/pkg/config"
	"pigate/pkg/database"
	"pigate/pkg/filehandler"
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
	repo, err := database.NewRepository(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database at %s: %v", cfg.DatabasePath, err)
	}
	defer repo.Close()

	// 4) Create GateController
	gateCtrl := gate.NewGateController(repo, cfg.GateOpenDuration)
	// Initialize the Raspberry Pi GPIO pin
	if err := gateCtrl.InitPinControl(cfg.GPIOPin); err != nil {
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

	// 7) Prepare the UpdateHandler
	ctx := context.Background()
	downloader, err := filehandler.NewS3Downloader(ctx, cfg.Location_ID)
	if err != nil {
		log.Fatalf("Failed to create S3 downloader: %v", err)
	}
	// Create the handler function that processes update notifications
	updateHandlerFunc := database.NewUpdateHandler(
		cfg.Location_ID,
		cfg.CredentialFileName,
		repo,       // Our SQLite repo
		downloader, // S3 downloader
	)

	// Subscribe to locationID/credentials/status
	client.SubscribeCredentialStatus(updateHandlerFunc)

	// Subscribe to locationID/pigate/command
	client.SubscribePigateCommand(gateCtrl.CommandHandler())

	// Keep main go routine running (non-busy)
	select {}
}
