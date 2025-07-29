package database_test

import (
	"context"
	"testing"
	"time"

	"pigate/pkg/database"
	"pigate/pkg/gate"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestGateManager(t *testing.T) database.GateManager {
	// Ensure deterministic behavior: treat local time as UTC in tests
	time.Local = time.UTC

	gm, err := database.NewSqliteGateManager("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open in-memory SQLite database: %v", err)
	}
	return gm
}

func TestGateManager(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize GateManager backed by in-memory SQLite
	gm := setupTestGateManager(t)
	defer gm.Close()

	// --- AccessManager tests ---
	code := "12345"
	cred := database.Credential{
		Code:        code,
		Username:    "test_user",
		AccessGroup: 1,
		LockedOut:   false,
		AutoUpdate:  false,
		OpenMode:    database.RegularOpen, // Test with RegularOpen
	}

	// PutCredential
	if err := gm.PutCredential(ctx, cred); err != nil {
		t.Fatalf("PutCredential failed: %v", err)
	}

	// GetCredential
	fetchedCred, err := gm.GetCredential(ctx, code)
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}
	if *fetchedCred != cred {
		t.Errorf("Fetched credential = %+v; want %+v", fetchedCred, cred)
	}

	// Test OpenMode: change to LockOpen and update
	cred.OpenMode = database.LockOpen
	if err := gm.PutCredential(ctx, cred); err != nil {
		t.Fatalf("PutCredential (LockOpen) failed: %v", err)
	}
	fetchedCred, err = gm.GetCredential(ctx, code)
	if err != nil {
		t.Fatalf("GetCredential (LockOpen) failed: %v", err)
	}
	if fetchedCred.OpenMode != database.LockOpen {
		t.Errorf("Fetched OpenMode = %v; want %v", fetchedCred.OpenMode, database.LockOpen)
	}

	// GetAllCredentials
	allCreds, err := gm.GetCredentials(ctx)
	if err != nil {
		t.Fatalf("GetAllCredentials failed: %v", err)
	}
	found := false
	for _, c := range allCreds {
		if c.Code == code && c.OpenMode == database.LockOpen {
			found = true
		}
	}
	if !found {
		t.Errorf("GetAllCredentials did not return credential with OpenMode=LockOpen")
	}

	// --- AccessTime tests ---
	accessTime := database.AccessTime{
		AccessGroup:  cred.AccessGroup,
		StartTime:    time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
		EndTime:      time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC),
		StartWeekday: time.Sunday,
		EndWeekday:   time.Saturday,
	}

	// PutAccessTime
	if err := gm.PutAccessTime(ctx, accessTime); err != nil {
		t.Fatalf("PutAccessTime failed: %v", err)
	}

	// GetAccessTime
	fetchedAt, err := gm.GetAccessTime(ctx, cred.AccessGroup)
	if err != nil {
		t.Fatalf("GetAccessTime failed: %v", err)
	}

	// Validate fields
	if fetchedAt.AccessGroup != accessTime.AccessGroup {
		t.Errorf("AccessGroup = %d; want %d", fetchedAt.AccessGroup, accessTime.AccessGroup)
	}
	if fetchedAt.StartTime.Format("15:04:05") != accessTime.StartTime.Format("15:04:05") {
		t.Errorf("StartTime = %s; want %s", fetchedAt.StartTime.Format("15:04:05"), accessTime.StartTime.Format("15:04:05"))
	}
	if fetchedAt.EndTime.Format("15:04:05") != accessTime.EndTime.Format("15:04:05") {
		t.Errorf("EndTime = %s; want %s", fetchedAt.EndTime.Format("15:04:05"), accessTime.EndTime.Format("15:04:05"))
	}
	if fetchedAt.StartWeekday != accessTime.StartWeekday {
		t.Errorf("StartWeekday = %v; want %v", fetchedAt.StartWeekday, accessTime.StartWeekday)
	}
	if fetchedAt.EndWeekday != accessTime.EndWeekday {
		t.Errorf("EndWeekday = %v; want %v", fetchedAt.EndWeekday, accessTime.EndWeekday)
	}

	// --- AccessLogger tests ---
	logEntry := database.GateLog{
		Code:   code,
		Time:   time.Now().UTC(),
		Status: database.StatusGranted,
	}

	// PutGateLog
	if err := gm.PutGateLog(ctx, logEntry); err != nil {
		t.Fatalf("PutGateLog failed: %v", err)
	}

	// GetGateLogs
	logs, err := gm.GetGateLogs(ctx)
	if err != nil {
		t.Fatalf("GetGateLogs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("GetGateLogs returned %d logs; want 1", len(logs))
	}
	if logs[0].Code != logEntry.Code || logs[0].Status != logEntry.Status {
		t.Errorf("GetGateLogs[0] = %+v; want Code=%s Status=%s", logs[0], logEntry.Code, logEntry.Status)
	}
	if logs[0].Time.Unix() != logEntry.Time.Unix() {
		t.Errorf("Log Time = %d; want %d", logs[0].Time.Unix(), logEntry.Time.Unix())
	}

	// --- GateController integration tests ---
	mockGateOpenDuration := 3
	controller := gate.NewGateController(gm, mockGateOpenDuration)
	defer controller.Close()

	// Valid case: 10:00 AM
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	if ok := controller.ValidateCredential(code, now); !ok {
		t.Errorf("Credential should be valid at %v", now)
	}

	// Before access window: 07:00 AM
	now = time.Date(2024, 1, 1, 7, 0, 0, 0, time.UTC)
	if ok := controller.ValidateCredential(code, now); ok {
		t.Errorf("Credential should NOT be valid at %v", now)
	}

	// After access window: 08:00 PM
	now = time.Date(2024, 1, 1, 20, 0, 0, 0, time.UTC)
	if ok := controller.ValidateCredential(code, now); ok {
		t.Errorf("Credential should NOT be valid at %v", now)
	}
}
