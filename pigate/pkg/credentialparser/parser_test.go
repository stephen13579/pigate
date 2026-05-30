package credentialparser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseCredentialFile(t *testing.T) {
	// Create a temporary CSV file
	tempFile, err := os.CreateTemp("", "test_credentials_*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write sample data
	csvData := `ACCOUNT,Resident,H,AAC,PHONE,DIR,ENT,SL,DEVICE#,FL,ER,NOTES,VENDOR
				DoorKing,William Henry,N,,,,,01,12345,,1,A17,N
				DoorKing,Stephen Thields,N,,,,,00,54321,,1,1,N`
	tempFile.WriteString(csvData)
	tempFile.Close()

	// Parse the file
	credentials, err := ParseCredentialFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Validate results
	expected := 2
	if len(credentials) != expected {
		t.Errorf("Expected %d credentials, got %d", expected, len(credentials))
	}

	// Check specific values
	if credentials[0].Username != "William Henry" || credentials[0].Code != "12345" || credentials[0].LockedOut != false {
		t.Errorf("Unexpected credential data: %+v", credentials[0])
	}
	if credentials[1].Username != "Stephen Thields" || credentials[1].Code != "54321" || credentials[1].LockedOut != true {
		t.Errorf("Unexpected credential data: %+v", credentials[1])
	}
}

func TestFindTextFileReturnsErrorWhenDirectoryHasNoTextFile(t *testing.T) {
	dir := t.TempDir()

	filePath, err := FindTextFile(dir)
	if !errors.Is(err, ErrCredentialTextFileNotFound) {
		t.Fatalf("FindTextFile() error = %v, want ErrCredentialTextFileNotFound", err)
	}
	if filePath != "" {
		t.Fatalf("FindTextFile() path = %q, want empty path", filePath)
	}
}

func TestFindTextFileFindsTextFile(t *testing.T) {
	dir := t.TempDir()
	expected := filepath.Join(dir, "GateCode.txt")
	if err := os.WriteFile(expected, []byte("ACCOUNT,Resident,SL,DEVICE#\n"), 0644); err != nil {
		t.Fatalf("failed to write text fixture: %v", err)
	}

	filePath, err := FindTextFile(dir)
	if err != nil {
		t.Fatalf("FindTextFile() error = %v", err)
	}
	if filePath != expected {
		t.Fatalf("FindTextFile() path = %q, want %q", filePath, expected)
	}
}
