package database

import (
	"context"
	"time"
)

type AccessManager interface {
	// Credential methods
	PutCredential(ctx context.Context, cred Credential) error
	GetCredential(ctx context.Context, code string) (*Credential, error)
	GetAllCredentials(ctx context.Context) ([]Credential, error)
	DeleteCredential(ctx context.Context, code string) error

	// AccessTime methods
	PutAccessTime(ctx context.Context, at AccessTime) error
	GetAccessTime(ctx context.Context, accessGroup int) (*AccessTime, error)
	DeleteAccessTime(ctx context.Context, accessGroup int) error
}

type AccessLogger interface {
	// Gate Logs
	PutGateLog(ctx context.Context, log GateLog) error
	GetGateLogs(ctx context.Context) ([]GateLog, error)
}

type Credential struct {
	Code        string
	Username    string
	AccessGroup int
	LockedOut   bool
}

// TODO: create default access time for users
type AccessTime struct {
	AccessGroup int
	StartTime   int
	EndTime     int
}

type GateLog struct {
	Code   string
	Time   time.Time
	Status string
}
