package database

import (
	"context"
	"log"
	"time"
)

func HandleUpdateNotification(repo AccessManager, tableName string) func(topic string, message string) {
	return func(topic string, message string) {
		log.Printf("Received update notification: %s", message)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := SyncCredentials(ctx, repo, tableName); err != nil {
			log.Printf("Sync failed: %v. Will retry later.", err)
		}
	}
}

func SyncCredentials(ctx context.Context, repo AccessManager, tableName string) error {
	backend, err := NewDynamoAccessManager(ctx, tableName)
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
	for _, cred := range credentials {
		if err := repo.PutCredential(ctx, cred); err != nil {
			log.Printf("Failed to sync credential %s: %v", cred.Code, err)
		}
	}

	log.Println("Credential sync completed successfully.")
	return nil
}
