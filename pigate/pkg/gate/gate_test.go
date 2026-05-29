//go:build cgo

package gate_test

import (
	"context"
	"testing"
	"time"

	"pigate/pkg/database"
	"pigate/pkg/gate"

	_ "github.com/mattn/go-sqlite3"
)

func TestValidateCredential(t *testing.T) {
	time.Local = time.UTC

	gm, err := database.NewSqliteGateManager("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open in-memory SQLite database: %v", err)
	}
	defer gm.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	code := "12345"
	cred := database.Credential{
		Code:        code,
		Username:    "test_user",
		AccessGroup: 1,
		LockedOut:   false,
		AutoUpdate:  false,
		OpenMode:    database.RegularOpen,
	}
	if err := gm.PutCredential(ctx, cred); err != nil {
		t.Fatalf("PutCredential failed: %v", err)
	}

	accessTime := database.AccessTime{
		AccessGroup:  cred.AccessGroup,
		StartTime:    time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
		EndTime:      time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC),
		StartWeekday: time.Sunday,
		EndWeekday:   time.Saturday,
	}
	if err := gm.PutAccessTime(ctx, accessTime); err != nil {
		t.Fatalf("PutAccessTime failed: %v", err)
	}

	controller := gate.NewGateController(gm, 3)
	defer controller.Close()

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			name: "valid during access window",
			now:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "invalid before access window",
			now:  time.Date(2024, 1, 1, 7, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "invalid after access window",
			now:  time.Date(2024, 1, 1, 20, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := controller.ValidateCredential(code, tt.now); got != tt.want {
				t.Fatalf("ValidateCredential() = %v, want %v", got, tt.want)
			}
		})
	}
}
