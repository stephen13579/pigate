package database_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"pigate/pkg/database"
)

// MockDownloader simulates S3Downloader behavior
type MockDownloader struct {
	Data []byte
	Err  error
}

func (m *MockDownloader) DownloadFileToMemory(ctx context.Context, key string) (*bytes.Buffer, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return bytes.NewBuffer(m.Data), nil
}

type MockRepository struct {
	Credentials []database.Credential
	AccessTimes []database.AccessTime
	GateLogs    []database.GateLog
	PutErr      error
	GetErr      error
	CloseErr    error
}

// Implement Close
func (m *MockRepository) Close() error {
	return m.CloseErr
}

// Implement PutCredential
func (m *MockRepository) PutCredential(cred database.Credential) error {
	if m.PutErr != nil {
		return m.PutErr
	}
	// Check if credential exists
	for i, c := range m.Credentials {
		if c.Code == cred.Code {
			m.Credentials[i] = cred // Update
			return nil
		}
	}
	m.Credentials = append(m.Credentials, cred) // Insert
	return nil
}

// Implement GetCredential
func (m *MockRepository) GetCredential(code string) (*database.Credential, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	for _, c := range m.Credentials {
		if c.Code == code {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("credential not found")
}

// Implement GetAccessTimes
func (m *MockRepository) GetCredentials() ([]database.Credential, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	return m.Credentials, nil
}

// Implement DeleteCredential
func (m *MockRepository) DeleteCredential(code string) error {
	return nil
}

// Implement PutAccessTime
func (m *MockRepository) PutAccessTime(at database.AccessTime) error {
	if m.PutErr != nil {
		return m.PutErr
	}
	// Check if access time exists
	for i, a := range m.AccessTimes {
		if a.AccessGroup == at.AccessGroup {
			m.AccessTimes[i] = at // Update
			return nil
		}
	}
	m.AccessTimes = append(m.AccessTimes, at) // Insert
	return nil
}

// Implement GetAccessTime
func (m *MockRepository) GetAccessTime(groupID int) (*database.AccessTime, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	for _, a := range m.AccessTimes {
		if a.AccessGroup == groupID {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("access time not found")
}

// Implement DeleteCredential
func (m *MockRepository) DeleteAccessTime(groupID int) error {
	return nil
}

// Implement AddGateLog
func (m *MockRepository) PutGateLog(log database.GateLog) error {
	if m.PutErr != nil {
		return m.PutErr
	}
	m.GateLogs = append(m.GateLogs, log)
	return nil
}

// Implement GetGateLogs
func (m *MockRepository) GetGateLogs() ([]database.GateLog, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	return m.GateLogs, nil
}

func TestHandleUpdateNotification(t *testing.T) {
	// Mock data
	mockCredentials := []database.Credential{
		{Code: "12345", Username: "user1", AccessGroup: 1, LockedOut: false},
		{Code: "67890", Username: "user2", AccessGroup: 2, LockedOut: true},
	}
	mockData, _ := json.Marshal(mockCredentials)

	// Create mocks
	mockDownloader := &MockDownloader{Data: mockData, Err: nil}
	mockRepo := &MockRepository{}

	// Create the UpdateHandler with mocks
	handlerFunc := database.NewUpdateHandler(mockRepo, mockDownloader)

	// Simulate MQTT message
	handlerFunc("nil", "nil")

	// Verify the results
	if len(mockRepo.Credentials) != len(mockCredentials) {
		t.Fatalf("Expected %d credentials to be Puted, got %d", len(mockCredentials), len(mockRepo.Credentials))
	}

	for i, cred := range mockRepo.Credentials {
		if cred != mockCredentials[i] {
			t.Errorf("Credential mismatch at index %d: got %+v, expected %+v", i, cred, mockCredentials[i])
		}
	}
}
