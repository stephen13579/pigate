package database

import (
	"context"
	"database/sql"
)

// -------------------------------------------------------------------
// AccessManager
// -------------------------------------------------------------------

// Ensure sqliteRepository implements AccessManager
var _ AccessManager = (*sqlitAccessManager)(nil)

type sqlitAccessManager struct {
	db *sql.DB
}

func NewAccessManager(db *sql.DB) (AccessManager, error) {
	logger := &sqlitAccessManager{db: db}
	if err := logger.createTables(); err != nil {
		return nil, err
	}
	return logger, nil
}

func (r *sqlitAccessManager) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS credentials (
			code TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			access_group INTEGER NOT NULL,
			locked_out BOOLEAN NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS access_times (
			access_group INTEGER PRIMARY KEY,
			start_time INTEGER NOT NULL,
			end_time INTEGER NOT NULL
		);`,
	}
	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (r *sqlitAccessManager) PutCredential(ctx context.Context, cred Credential) error {
	query := `
		INSERT INTO credentials (code, username, access_group, locked_out)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET
			username = excluded.username,
			access_group = excluded.access_group,
			locked_out = excluded.locked_out`
	_, err := r.db.Exec(query, cred.Code, cred.Username, cred.AccessGroup, cred.LockedOut)
	return err
}

func (r *sqlitAccessManager) GetCredential(ctx context.Context, code string) (*Credential, error) {
	query := `SELECT code, username, access_group, locked_out FROM credentials WHERE code = ?`
	var c Credential
	err := r.db.QueryRow(query, code).Scan(&c.Code, &c.Username, &c.AccessGroup, &c.LockedOut)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *sqlitAccessManager) GetAllCredentials(ctx context.Context) ([]Credential, error) {
	query := `SELECT code, username, access_group, locked_out FROM credentials`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []Credential
	for rows.Next() {
		var cred Credential
		err := rows.Scan(&cred.Code, &cred.Username, &cred.AccessGroup, &cred.LockedOut)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, cred)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return credentials, nil
}

func (r *sqlitAccessManager) DeleteCredential(ctx context.Context, code string) error {
	// Implementation for deleting a credential
	return nil
}

func (r *sqlitAccessManager) PutAccessTime(ctx context.Context, at AccessTime) error {
	query := `
		INSERT INTO access_times (access_group, start_time, end_time)
		VALUES (?, ?, ?)
		ON CONFLICT(access_group) DO UPDATE SET
			start_time = excluded.start_time,
			end_time = excluded.end_time`
	_, err := r.db.Exec(query, at.AccessGroup, at.StartTime, at.EndTime)
	return err
}

func (r *sqlitAccessManager) GetAccessTime(gctx context.Context, groupID int) (*AccessTime, error) {
	query := `SELECT access_group, start_time, end_time FROM access_times WHERE access_group = ?`
	var at AccessTime
	err := r.db.QueryRow(query, groupID).Scan(&at.AccessGroup, &at.StartTime, &at.EndTime)
	if err != nil {
		return nil, err
	}
	return &at, nil
}

func (r *sqlitAccessManager) DeleteAccessTime(ctx context.Context, groupID int) error {
	// Implementation for deleting an access time
	return nil
}

// -------------------------------------------------------------------
// AccessLogger
// -------------------------------------------------------------------

// Ensure sqliteRepository implements AccessLogger
var _ AccessLogger = (*sqliteAccessLogger)(nil)

type sqliteAccessLogger struct {
	db *sql.DB
}

func NewAccessLogger(db *sql.DB) (AccessLogger, error) {
	logger := &sqliteAccessLogger{db: db}
	if err := logger.createTables(); err != nil {
		return nil, err
	}
	return logger, nil
}

func (r *sqliteAccessLogger) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS gate_request_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL,
			time DATETIME NOT NULL,
			status TEXT NOT NULL
		);`,
	}
	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (r *sqliteAccessLogger) PutGateLog(ctx context.Context, logEntry GateLog) error {
	query := `INSERT INTO gate_request_log (code, time, status) VALUES (?, ?, ?)`
	_, err := r.db.Exec(query, logEntry.Code, logEntry.Time, logEntry.Status)
	return err
}

func (r *sqliteAccessLogger) GetGateLogs(ctx context.Context) ([]GateLog, error) {
	query := `SELECT code, time, status FROM gate_request_log`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []GateLog
	for rows.Next() {
		var log GateLog
		if err := rows.Scan(&log.Code, &log.Time, &log.Status); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}
