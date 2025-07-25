package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"pigate/pkg/config"
	"pigate/pkg/credentialparser"
	"pigate/pkg/messenger"
)

const application string = "credentialserver"
const FILENAME = "credentials.json"

var (
	lastHandleTime time.Time
	handleCooldown = 5 * time.Second
	handleMu       sync.Mutex
)

func main() {
	// 1) Parse command-line flags for config path
	var configFilePath string
	flag.StringVar(&configFilePath, "c", "/workspace/pigate/pkg/config",
		"Path to the configuration file")
	flag.Parse()

	// 2) Load configuration for credentialserver
	cfg := config.LoadConfig(configFilePath, application+"-config").(*config.CredentialServerConfig)

	// 3) Create messenger
	client := messenger.NewMQTTClient(cfg.MQTTBroker, application, cfg.Location_ID)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect to MQTT broker (%s): %v", cfg.MQTTBroker, err)
	}
	defer client.Disconnect()

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name)

	// 4) Parse credential file
	filePath, err := credentialparser.FindTextFile(cfg.FileWatcherPath)
	if err != nil {
		fmt.Printf("Did not find credential file on startup, this is fine.")
	} else {
		err := credentialparser.HandleFile(filePath, connStr)
		if err != nil {
			fmt.Printf("failed to handle file update: %s", err)
		} else {
			// Send mqtt message that an update is available
			client.NotifyNewCredentials()
		}
	}

	// 5) Start FileWatcher for credential file
	fileWatcher := credentialparser.NewFileWatcher(cfg.FileWatcherPath, func(filePath string) {
		err := credentialparser.HandleFile(filePath, connStr)
		if err != nil {
			fmt.Printf("failed to handle file update: %s", err)
		} else {
			// Send mqtt message that an update is available
			client.NotifyNewCredentials()
		}
	})

	go func() {
		err := fileWatcher.Start()
		if err != nil {
			log.Fatalf("file watcher failed: %v", err)
		}
	}()

	// Non-busy infinite loop
	select {}
}
