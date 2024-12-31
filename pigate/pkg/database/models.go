package database

import (
	"time"
)

type Repository interface {
	Close() error

	// Credential methods
	UpsertCredential(cred Credential) error
	GetCredential(code string) (*Credential, error)

	// AccessTime methods
	UpsertAccessTime(at AccessTime) error
	GetAccessTime(groupID int) (*AccessTime, error)

	// Gate Logs
	AddGateLog(log GateLog) error
	GetGateLogs() ([]GateLog, error)
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
