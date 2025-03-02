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
	"time"

	"pigate/pkg/database"
)

const FILENAME = "credentials.json"

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

	// Extract the header and find column indices
	header := records[0]
	usernameIndex, codeIndex, lockedOutIndex := -1, -1, -1
	for i, column := range header {
		switch column {
		case "Resident":
			usernameIndex = i
		case "DEVICE#":
			codeIndex = i
		case "SL":
			lockedOutIndex = i
		}
	}
	if usernameIndex == -1 || codeIndex == -1 || lockedOutIndex == -1 {
		fmt.Println("Required columns not found in the CSV file.")
		return nil, nil
	}

	// Parse the CSV rows into the Credential struct
	var entries []database.Credential
	for _, record := range records[1:] { // Skip the header row
		lockedOut := record[lockedOutIndex] == "00" // "00" means locked out
		entry := database.Credential{
			Username:    record[usernameIndex],
			Code:        record[codeIndex],
			LockedOut:   lockedOut,
			AccessGroup: 1, // TODO: making access group 1 default, probably a better way to handle this than a magic number
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func HandleFile(filePath, table string) error {
	// Parse the CSV file
	credentials, err := ParseCredentialFile(filePath)
	if err != nil {
		log.Printf("Failed to parse CSV file %s: %v", filePath, err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Upload to DynamoDB
	dynamoDB, err := database.NewDynamoAccessManager(ctx, table)
	if err != nil {
		log.Printf("Failed to access DynamDB: %v", err)
		return err
	}

	err = dynamoDB.PutCredentials(ctx, credentials)
	if err != nil {
		log.Printf("Failed to put items to dynamoDB: %v", err)
		return err
	}

	log.Printf("Using file: %s, succesfully updated remote database.", filePath)

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
