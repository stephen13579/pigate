package database

import (
	"context"
)

type GateManager interface {
	AccessManager
	AccessLogger
	// Close closes the underlying database connection.
	Close() error
}

type AccessManager interface {
	// Credential methods
	PutCredential(ctx context.Context, cred Credential) error
	PutCredentials(ctx context.Context, creds []Credential) error
	GetCredential(ctx context.Context, code string) (*Credential, error)
	GetCredentials(ctx context.Context) ([]Credential, error)
	DeleteCredential(ctx context.Context, code string) error
	DeleteCredentials(ctx context.Context, codes []string) error

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
