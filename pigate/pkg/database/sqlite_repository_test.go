package database_test

import (
	"context"
	"pigate/pkg/database"
	"pigate/pkg/gate"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestRepository(t *testing.T) *database.Repository {
	repo, err := database.NewRepository(":memory:") // Use SQLite in-memory DB for testing
	if err != nil {
		t.Fatalf("failed to create Repository: %v", err)
	}
	return repo
}

func TestGateControllerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set up repository (now using an in-memory DB)
	repo := setupTestRepository(t)
	defer repo.Close()

	// Test AccessManager Functions
	code := "12345"
	credential := database.Credential{
		Code:        code,
		Username:    "test_user",
		AccessGroup: 1,
		LockedOut:   false,
	}

	// Test PutCredential
	if err := repo.AccessMgr.PutCredential(ctx, credential); err != nil {
		t.Fatalf("PutCredential failed: %v", err)
	}

	// Test GetCredential
	fetchedCredential, err := repo.AccessMgr.GetCredential(ctx, code)
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}
	if *fetchedCredential != credential {
		t.Errorf("Fetched credential does not match: got %+v, want %+v", fetchedCredential, credential)
	}

	// Test GetAllCredentials
	allCredentials, err := repo.AccessMgr.GetAllCredentials(ctx)
	if err != nil {
		t.Fatalf("GetAllCredentials failed: %v", err)
	}
	if len(allCredentials) != 1 || allCredentials[0] != credential {
		t.Errorf("GetAllCredentials returned unexpected results: got %+v, want %+v", allCredentials, []database.Credential{credential})
	}

	// Test AccessTime Functions
	accessTime := database.AccessTime{
		AccessGroup: credential.AccessGroup,
		StartTime:   540,  // 9:00 AM
		EndTime:     1020, // 5:00 PM
	}

	// Test PutAccessTime
	if err := repo.AccessMgr.PutAccessTime(ctx, accessTime); err != nil {
		t.Fatalf("PutAccessTime failed: %v", err)
	}

	// Test GetAccessTime
	fetchedAccessTime, err := repo.AccessMgr.GetAccessTime(ctx, credential.AccessGroup)
	if err != nil {
		t.Fatalf("GetAccessTime failed: %v", err)
	}
	if *fetchedAccessTime != accessTime {
		t.Errorf("Fetched access time does not match: got %+v, want %+v", fetchedAccessTime, accessTime)
	}

	// Test AccessLogger Functions
	logEntry := database.GateLog{
		Code:   code,
		Time:   time.Now(),
		Status: "Success",
	}

	// Test PutGateLog
	if err := repo.AccessLogger.PutGateLog(ctx, logEntry); err != nil {
		t.Fatalf("PutGateLog failed: %v", err)
	}

	// Test GetGateLogs
	allLogs, err := repo.AccessLogger.GetGateLogs(ctx)
	if err != nil {
		t.Fatalf("GetGateLogs failed: %v", err)
	}
	if len(allLogs) != 1 || allLogs[0].Code != logEntry.Code || allLogs[0].Status != logEntry.Status {
		t.Errorf("GetGateLogs returned unexpected results: got %+v, want %+v", allLogs, []database.GateLog{logEntry})
	}

	// Test GateController Integration (Now Using Repository)
	mockGateOpenDuration := 3
	controller := gate.NewGateController(repo, mockGateOpenDuration)
	defer controller.Close()

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
