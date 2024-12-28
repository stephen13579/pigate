package credentialparser

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Credential struct {
	Username  string `json:"username"`
	Code      string `json:"code"`
	LockedOut bool   `json:"locked_out"`
}

func ParseCredentialFile(filepath string) ([]Credential, error) {
	// Open the CSV file
	file, err := os.Open(filepath)
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
	var entries []Credential
	for _, record := range records[1:] { // Skip the header row
		lockedOut := record[lockedOutIndex] == "00" // "00" means locked out
		entry := Credential{
			Username:  record[usernameIndex],
			Code:      record[codeIndex],
			LockedOut: lockedOut,
		}
		entries = append(entries, entry)
	}

	// Debug output (optional)
	for _, entry := range entries {
		fmt.Printf("Name: %s, Gate Code: %s, Locked Out: %t\n", entry.Username, entry.Code, entry.LockedOut)
	}

	return entries, nil
}

func createJSONFile(outputDir string, credentials []Credential) (string, error) {
	// Generate a filename with a timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s/credentials_%s.json", outputDir, timestamp)

	// Create the JSON file
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("error creating JSON file: %w", err)
	}
	defer file.Close()

	// Marshal the credentials slice into JSON
	jsonData, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling credentials to JSON: %w", err)
	}

	// Write the JSON data to the file
	_, err = file.Write(jsonData)
	if err != nil {
		return "", fmt.Errorf("error writing to JSON file: %w", err)
	}

	fmt.Printf("JSON file created successfully: %s\n", filename)
	return filename, nil
}
