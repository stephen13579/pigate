package database_test

import (
	"context"
	"database/sql"
	"pigate/pkg/database"
	"pigate/pkg/gate"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:") // Use in-memory SQLite for testing
	if err != nil {
		t.Fatalf("Failed to open in-memory SQLite database: %v", err)
	}
	return db
}

func setupTestRepositories(t *testing.T, db *sql.DB) (database.AccessManager, database.AccessLogger) {
	// Create repositories
	accessManager, err := database.NewAccessManager(db)
	if err != nil {
		t.Fatalf("failed to create AccessManager: %v", err)
	}
	accessLogger, err := database.NewAccessLogger(db)
	if err != nil {
		t.Fatalf("failed to create AccessLogger: %v", err)
	}

	return accessManager, accessLogger
}

func TestGateControllerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := setupTestDB(t)
	defer db.Close()

	// Set up repositories
	accessManager, accessLogger := setupTestRepositories(t, db)

	// Test AccessManager Functions
	code := "12345"
	credential := database.Credential{
		Code:        code,
		Username:    "test_user",
		AccessGroup: 1,
		LockedOut:   false,
	}

	// Test PutCredential
	if err := accessManager.PutCredential(ctx, credential); err != nil {
		t.Fatalf("PutCredential failed: %v", err)
	}

	// Test GetCredential
	fetchedCredential, err := accessManager.GetCredential(ctx, code)
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}
	if *fetchedCredential != credential {
		t.Errorf("Fetched credential does not match: got %+v, want %+v", fetchedCredential, credential)
	}

	// Test GetCredentials
	allCredentials, err := accessManager.GetAllCredentials(ctx)
	if err != nil {
		t.Fatalf("GetCredentials failed: %v", err)
	}
	if len(allCredentials) != 1 || allCredentials[0] != credential {
		t.Errorf("GetCredentials returned unexpected results: got %+v, want %+v", allCredentials, []database.Credential{credential})
	}

	// Test AccessTime Functions
	accessTime := database.AccessTime{
		AccessGroup: credential.AccessGroup,
		StartTime:   540,  // 9:00 AM
		EndTime:     1020, // 5:00 PM
	}

	// Test PutAccessTime
	if err := accessManager.PutAccessTime(ctx, accessTime); err != nil {
		t.Fatalf("PutAccessTime failed: %v", err)
	}

	// Test GetAccessTime
	fetchedAccessTime, err := accessManager.GetAccessTime(ctx, credential.AccessGroup)
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
	if err := accessLogger.PutGateLog(ctx, logEntry); err != nil {
		t.Fatalf("PutGateLog failed: %v", err)
	}

	// Test GetGateLogs
	allLogs, err := accessLogger.GetGateLogs(ctx)
	if err != nil {
		t.Fatalf("GetGateLogs failed: %v", err)
	}
	if len(allLogs) != 1 || allLogs[0].Code != logEntry.Code || allLogs[0].Status != logEntry.Status {
		t.Errorf("GetGateLogs returned unexpected results: got %+v, want %+v", allLogs, []database.GateLog{logEntry})
	}

	// Test GateController Integration
	mockGateOpenDuration := 3
	controller := gate.NewGateController(accessManager, mockGateOpenDuration)
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
