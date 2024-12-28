package database

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Use in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory SQLite database: %v", err)
	}

	err = createTables(db)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	return db
}

func TestAddAndGetAccessTime(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Prepare test data
	startTime := 540 // 9:00 AM
	endTime := 1020  // 5:00 PM
	accessTime := AccessTime{
		AccessGroup: 1,
		StartTime:   startTime,
		EndTime:     endTime,
	}

	err := AddAccessTime(db, accessTime)
	if err != nil {
		t.Fatalf("Failed to add access time: %v", err)
	}

	retrieved, err := getAccessTime(db, accessTime.AccessGroup)
	if err != nil {
		t.Fatalf("Failed to retrieve access time: %v", err)
	}

	if retrieved.StartTime != accessTime.StartTime ||
		retrieved.EndTime != accessTime.EndTime ||
		retrieved.AccessGroup != accessTime.AccessGroup {
		t.Errorf("Retrieved access time does not match. Got: %+v, Expected: %+v", retrieved, accessTime)
	}
}

func TestAddAndGetCredential(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Prepare test data
	credential := Credential{
		Code:        "12345",
		Username:    "test_user",
		AccessGroup: 1,
		LockedOut:   false,
	}

	err := AddCredential(db, credential)
	if err != nil {
		t.Fatalf("Failed to add credential: %v", err)
	}

	retrieved, err := getCredential(db, credential.Code)
	if err != nil {
		t.Fatalf("Failed to retrieve credential: %v", err)
	}

	if retrieved.Code != credential.Code ||
		retrieved.Username != credential.Username ||
		retrieved.AccessGroup != credential.AccessGroup ||
		retrieved.LockedOut != credential.LockedOut {
		t.Errorf("Retrieved credential does not match. Got: %+v, Expected: %+v", retrieved, credential)
	}
}

func TestValidateCredential(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Prepare test data
	code := "12345"
	credential := Credential{
		Code:        code,
		Username:    "test_user",
		AccessGroup: 2,
		LockedOut:   false,
	}

	err := AddCredential(db, credential)
	if err != nil {
		t.Fatalf("Failed to add credential: %v", err)
	}

	startTime := 480 // 8:00 AM
	endTime := 1080  // 6:00 PM
	accessTime := AccessTime{
		AccessGroup: credential.AccessGroup,
		StartTime:   startTime,
		EndTime:     endTime,
	}
	err = AddAccessTime(db, accessTime)
	if err != nil {
		t.Fatalf("Failed to add access time: %v", err)
	}

	// Validate credential within valid time
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC) // 10:00 AM
	isValid, err := ValidateCredential(db, code, now)
	if !isValid || err != nil {
		t.Errorf("Credential should be valid during access time. Current time: %s, Access Time: %+v", now.Format("15:04:05"), accessTime)
	}

	// Validate credential before valid access time
	now = time.Date(2024, 1, 1, 7, 0, 0, 0, time.UTC) // 7:00 AM
	isValid, err = ValidateCredential(db, code, now)
	if isValid || err != nil {
		t.Errorf("Credential should not be valid outside access time. Current time: %s, Access Time: %+v", now.Format("15:04:05"), accessTime)
	}

	// Validate credential after valid access time
	now = time.Date(2024, 1, 1, 20, 0, 0, 0, time.UTC) // 7:00 AM
	isValid, err = ValidateCredential(db, code, now)
	if isValid || err != nil {
		t.Errorf("Credential should not be valid outside access time. Current time: %s, Access Time: %+v", now.Format("15:04:05"), accessTime)
	}
}

// test gate logging
func TestLogGateRequest(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Log gate request for specific time
	gateCommandRequest := GateRequestLog{
		Code:   "12345",
		Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		Status: "success",
	}

	err := AddGateRequestLog(db, gateCommandRequest)
	if err != nil {
		t.Fatalf("Failed to log gate request: %v", err)
	}

	// Retrieve gate request logs
	logs, err := GetGateRequestLogs(db)
	if err != nil {
		t.Fatalf("Failed to retrieve gate request logs: %v", err)
	}

	// Compare to expected values
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}
	if logs[0].Code != gateCommandRequest.Code ||
		logs[0].Status != gateCommandRequest.Status ||
		logs[0].Time != gateCommandRequest.Time {
		t.Errorf("Retrieved gate request log does not match. Got: %+v, Expected: %+v", logs[0], gateCommandRequest)
	}
}
