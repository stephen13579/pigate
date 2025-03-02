package database

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
)

const FILENAME = "crednetials.json"

// Downloader defines the interface for S3 download operations
type Downloader interface {
	DownloadFileToMemory(ctx context.Context, key string) (*bytes.Buffer, error)
}

type UpdateHandler struct {
	Key           string
	AccessManager AccessManager
	Downloader    Downloader
}

// NewUpdateHandler is the factory function that initializes and returns the handler
func NewUpdateHandler(manager AccessManager, downloader Downloader) func(string, string) {
	handler := &UpdateHandler{
		Key:           FILENAME,
		AccessManager: manager,
		Downloader:    downloader,
	}
	return handler.HandleUpdateNotification
}

// HandleUpdateNotification processes the MQTT message and updates credentials
func (h *UpdateHandler) HandleUpdateNotification(topic string, msg string) {
	log.Println("Received update notification via MQTT")

	ctx := context.Background()

	err := fetchAndUpdateCredentials(ctx, h.Key, h.AccessManager, h.Downloader)
	if err != nil {
		log.Printf("Error updating credentials: %v", err)
	}
}

// FetchAndUpdateCredentials fetches credentials from S3 and updates the database
func fetchAndUpdateCredentials(ctx context.Context, key string, repo AccessManager, downloader Downloader) error {
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
		if err := repo.PutCredential(ctx, cred); err != nil {
			return err
		}
	}

	log.Println("Successfully updated credentials")
	return nil
}
