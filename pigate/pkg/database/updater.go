package database

import (
	"bytes"
	"context"
	"encoding/json"
	"log"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// Downloader defines the interface for S3 download operations
type Downloader interface {
	DownloadFileToMemory(ctx context.Context, key string) (*bytes.Buffer, error)
}

type UpdateHandler struct {
	BucketName string
	Key        string
	Repository Repository
	Downloader Downloader
}

// NewUpdateHandler is the factory function that initializes and returns the handler
func NewUpdateHandler(bucketName, key string, repo Repository, downloader Downloader) func(string, MQTT.Message) {
	handler := &UpdateHandler{
		BucketName: bucketName,
		Key:        key,
		Repository: repo,
		Downloader: downloader,
	}
	return handler.HandleUpdateNotification
}

// HandleUpdateNotification processes the MQTT message and updates credentials
func (h *UpdateHandler) HandleUpdateNotification(topic string, msg MQTT.Message) {
	log.Println("Received update notification via MQTT")

	ctx := context.Background()

	err := FetchAndUpdateCredentials(ctx, h.BucketName, h.Key, h.Repository, h.Downloader)
	if err != nil {
		log.Printf("Error updating credentials: %v", err)
	}
}

// FetchAndUpdateCredentials fetches credentials from S3 and updates the database
func FetchAndUpdateCredentials(ctx context.Context, bucketName, key string, repo Repository, downloader Downloader) error {
	// Download credentials file
	buf, err := downloader.DownloadFileToMemory(ctx, key)
	if err != nil {
		return err
	}

	var credentials []Credential
	if err := json.Unmarshal(buf.Bytes(), &credentials); err != nil {
		return err
	}

	for _, cred := range credentials {
		if err := repo.UpsertCredential(cred); err != nil {
			return err
		}
	}

	log.Println("Successfully updated credentials")
	return nil
}
