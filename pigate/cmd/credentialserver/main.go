package main

import (
	"context"
	"flag"
	"log"
	"time"

	"pigate/pkg/config"
	"pigate/pkg/credentialparser"
)

const application string = "credentialserver"

func handleFile(filePath string, cfg *config.Config) {
	// Parse the CSV file
	credentials, err := credentialparser.ParseCredentialFile(filePath)
	if err != nil {
		log.Printf("Failed to parse CSV file %s: %v", filePath, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // Ensure resources are cleaned up

	// Upload to S3
	uploader, err := s3interface.NewS3Uploader(ctx, cfg.S3Bucket)
	if err != nil {
		log.Printf("Failed to initialize S3Uploader: %v", err)
		return
	}

	_, err = uploader.UploadJSON(ctx, credentials, "credentials")
	if err != nil {
		log.Printf("Failed to upload file to S3: %v", err)
		return
	}

	log.Printf("File %s successfully uploaded to S3", filePath)

	// Publish MQTT message

}

func main() {
	// 1) Parse command-line flags for config path
	var configFilePath string
	flag.StringVar(&configFilePath, "c", "/workspace/pigate/pkg/config",
		"Path to the configuration file")
	flag.Parse()

	// 2) Load configuration for gatecontroller
	cfg := config.LoadConfig(configFilePath, application+"-config").(config.CredentialServerConfig)

	// Start FileWatcher
	fileWatcher := credentialparser.NewFileWatcher("/", func(filePath string) {
		handleFile(filePath, cfg)
	})

	// Start file watcher in a goroutine
	go func() {
		err := fileWatcher.Start()
		if err != nil {
			log.Fatalf("File watcher error: %v", err)
		}
	}()

	// Non-busy infinite loop
	select {}
}
