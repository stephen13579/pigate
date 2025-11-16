package credentialparser

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pigate/pkg/database"
)

const FILENAME = "credentials.json"
const defaultAccessGroup = 0

func ParseCredentialFile(filePath string) ([]database.Credential, error) {
	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading CSV file:", err)
		return nil, err
	}
	if len(records) < 2 {
		fmt.Println("CSV file is empty or missing header.")
		return nil, nil
	}

	// Find column indices
	header := records[0]
	usernameIndex, codeIndex, lockedOutIndex := -1, -1, -1
	for i, col := range header {
		switch col {
		case "Resident":
			usernameIndex = i
		case "DEVICE#":
			codeIndex = i
		case "SL":
			lockedOutIndex = i
		}
	}
	if usernameIndex < 0 || codeIndex < 0 || lockedOutIndex < 0 {
		fmt.Println("Required columns not found in the CSV file.")
		return nil, nil
	}

	var entries []database.Credential
	seen := make(map[string]struct{}, len(records))

	// Process rows
	for _, record := range records[1:] {
		code := strings.TrimSpace(record[codeIndex])
		if code == "" {
			continue // skip empty codes
		}
		if _, dup := seen[code]; dup {
			continue // skip duplicates
		}
		seen[code] = struct{}{}

		lockedOut := record[lockedOutIndex] == "00" // "00" means locked out
		entry := database.Credential{
			Code:        code,
			Username:    record[usernameIndex],
			AccessGroup: defaultAccessGroup,
			LockedOut:   lockedOut,
			AutoUpdate:  true, // this record comes from the external feed
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// HandleFile will:
//  1. Parse the CSV at filePath
//  2. Load remote credentials (filtering on AutoUpdate)
//  3. Delete any remote-only AutoUpdate creds
//  4. Upsert all file credentials
func HandleFile(filePath, connStr string) error {
	// --- 1) Parse the CSV file ----------------------------------------
	fileCredentials, err := ParseCredentialFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse CSV file %s: %w", filePath, err)
	}

	// --- 2) Create repo & load remote creds ---------------------------
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo, err := database.NewPostgresAccessManager(ctx, connStr)
	if err != nil {
		return fmt.Errorf("new repo: %w", err)
	}

	remoteAll, err := repo.GetCredentials(ctx)
	if err != nil {
		return fmt.Errorf("get remote credentials: %w", err)
	}

	// Filter only AutoUpdate‐marked remotes, build map by Code
	remoteMap := make(map[string]database.Credential, len(remoteAll))
	for _, r := range remoteAll {
		if r.AutoUpdate {
			remoteMap[r.Code] = r
		}
	}

	// --- 3) Build file map & upsert list ------------------------------
	fileMap := make(map[string]database.Credential, len(fileCredentials))
	toUpsert := make([]database.Credential, 0, len(fileCredentials))
	for _, f := range fileCredentials {
		fileMap[f.Code] = f
		toUpsert = append(toUpsert, f)
	}

	// --- 4) Compute delete list (remote-only codes) -------------------
	toDelete := make([]string, 0, len(remoteMap))
	for code := range remoteMap {
		if _, found := fileMap[code]; !found {
			toDelete = append(toDelete, code)
		}
	}

	// --- 5) Delete old creds ------------------------------------------
	if len(toDelete) > 0 {
		log.Printf("Removing %d old credentials: %v", len(toDelete), toDelete)
		if err := repo.DeleteCredentials(ctx, toDelete); err != nil {
			return fmt.Errorf("delete old credentials: %w", err)
		}
	}

	// --- 6) Upsert new/updated creds ----------------------------------
	if len(toUpsert) > 0 {
		log.Printf("Upserting %d credentials", len(toUpsert))
		if err := repo.PutCredentials(ctx, toUpsert); err != nil {
			return fmt.Errorf("put new credentials: %w", err)
		}
	}

	log.Printf("Successfully synced %d credentials from %s --> remote DB",
		len(toUpsert), filePath)
	return nil
}

// Find .txt file in directory path
func FindTextFile(dirPath string) (string, error) {
	var txtFilePath string
	err := ensureFileExists(dirPath)
	if err != nil {
		return txtFilePath, fmt.Errorf("error ensuring file exists: %w", err)
	}

	// Look for a .txt file in the credentialDataPath directory
	err = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".txt" {
			txtFilePath = path
			return io.EOF // Stop walking after finding the first .txt file
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return txtFilePath, fmt.Errorf("error searching for .txt file: %w", err)
	}
	return txtFilePath, nil
}

// ensureFileExists ensures the file exists
func ensureFileExists(path string) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
	}
	return nil
}
