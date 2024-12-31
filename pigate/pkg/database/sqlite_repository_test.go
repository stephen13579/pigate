package database_test

import (
	"pigate/pkg/database"
	"pigate/pkg/gate"

	"testing"
	"time"
)

// setupTestRepo creates an in-memory SQLite repository (":memory:") for testing
func setupTestRepo(t *testing.T) database.Repository {
	repo, err := database.NewRepository(":memory:") // uses SQLite in-memory DB
	if err != nil {
		t.Fatalf("Failed to open in-memory SQLite database: %v", err)
	}
	return repo
}

// TestGateControllerIntegration tests GateController integration with Repository
func TestGateControllerIntegration(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	// Initialize GateController
	mockGateOpenDuration := 3
	controller := gate.NewGateController(repo, mockGateOpenDuration)
	defer controller.Close()

	// Setup credential and access time
	code := "12345"
	credential := database.Credential{
		Code:        code,
		Username:    "test_user",
		AccessGroup: 1,
		LockedOut:   false,
	}
	if err := repo.UpsertCredential(credential); err != nil {
		t.Fatalf("Failed to add credential: %v", err)
	}

	accessTime := database.AccessTime{
		AccessGroup: credential.AccessGroup,
		StartTime:   540,  // 9:00 AM
		EndTime:     1020, // 5:00 PM
	}
	if err := repo.UpsertAccessTime(accessTime); err != nil {
		t.Fatalf("Failed to add access time: %v", err)
	}

	// Validate within access time
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC) // 10:00 AM
	isValid := controller.ValidateCredential(code, now)
	if !isValid {
		t.Errorf("Credential should be valid at %v", now)
	}

	// Validate before access time
	now = time.Date(2024, 1, 1, 7, 0, 0, 0, time.UTC) // 7:00 AM
	isValid = controller.ValidateCredential(code, now)
	if isValid {
		t.Errorf("Credential should NOT be valid at %v", now)
	}

	// Validate after access time
	now = time.Date(2024, 1, 1, 20, 0, 0, 0, time.UTC) // 8:00 PM
	isValid = controller.ValidateCredential(code, now)
	if isValid {
		t.Errorf("Credential should NOT be valid at %v", now)
	}
}

func TestGateControllerLogRequest(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	// Initialize GateController
	mockGateOpenDuration := 3
	controller := gate.NewGateController(repo, mockGateOpenDuration)
	defer controller.Close()

	// Log gate request
	code := "12345"
	gateLog := database.GateLog{
		Code:   code,
		Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		Status: "success",
	}
	if err := repo.AddGateLog(gateLog); err != nil {
		t.Fatalf("Failed to log gate request: %v", err)
	}

	// Retrieve gate logs
	logs, err := repo.GetGateLogs()
	if err != nil {
		t.Fatalf("Failed to retrieve gate logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}
	got := logs[0]
	if got.Code != gateLog.Code || got.Status != gateLog.Status || got.Time != gateLog.Time {
		t.Errorf("GateLog mismatch.\nGot:      %+v\nExpected: %+v", got, gateLog)
	}
}
