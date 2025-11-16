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

		if err := SyncAccessTimes(ctx, access, connStr); err != nil {
			log.Printf("Access time sync failed: %v. Will retry later.", err)
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

func SyncAccessTimes(ctx context.Context, access AccessManager, connStr string) error {
	backend, err := NewPostgresAccessManager(ctx, connStr)
	if err != nil {
		log.Printf("Failed to create new instance of AccessManager: %v", err)
		return err
	}

	log.Println("Syncing access times from AccessManager...")

	// Fetch access times from AccessManager
	accessTimes, err := backend.GetAccessTimes(ctx)
	if err != nil {
		log.Printf("Failed to fetch access times: %v", err)
		return err
	}

	// Store in local database
	for _, at := range accessTimes {
		if err := access.PutAccessTime(ctx, at); err != nil {
			log.Printf("Failed to sync access time for group %d: %v", at.AccessGroup, err)
		}
	}

	log.Println("Access time sync completed successfully.")
	return nil
}
