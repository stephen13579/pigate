package main

import (
	"context"
	"flag"
	"log"
	"pigate/pkg/config"
	"pigate/pkg/credentialparser"
	"pigate/pkg/s3interface"
	"time"
)

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
	var configFilePath string

	flag.StringVar(&configFilePath, "c", "/workspace/pigate/pkg/config", "Please provide file path for this programs config file") // TODO make this default filepath make more sense

	flag.Parse()

	// Load configurations
	cfg := config.LoadConfig(configFilePath)

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
