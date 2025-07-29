package database

import (
	"context"
	"log"
	"time"
)

func HandleUpdateNotification(access AccessManager, connStr string) func(topic string, message string) {
	return func(topic string, message string) {
		log.Printf("Received update notification: %s", message)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := SyncCredentials(ctx, access, connStr); err != nil {
			log.Printf("Sync failed: %v. Will retry later.", err)
		}
	}
}

func SyncCredentials(ctx context.Context, access AccessManager, connStr string) error {
	backend, err := NewPostgresAccessManager(ctx, connStr)
	if err != nil {
		log.Printf("Failed to create new instance of AccessManager: %v", err)
		return err
	}

	log.Println("Syncing credentials from AccessManager...")

	// Fetch credentials from AccessManager
	credentials, err := backend.GetCredentials(ctx)
	if err != nil {
		log.Printf("Failed to fetch credentials: %v", err)
		return err
	}

	// Store in local database
	if err := access.PutCredentials(ctx, credentials); err != nil {
		log.Printf("Failed to sync credentials: %v", err)
	}

	log.Println("Credential sync completed successfully.")
	return nil
}
