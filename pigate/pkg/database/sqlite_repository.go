package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// Ensure sqliteRepository implements Repository at compile time
var _ Repository = (*sqliteRepository)(nil)

type sqliteRepository struct {
	db *sql.DB
}

// opens a DB connection and sets up tables if they do not exist
func NewRepository(dbPath string) (Repository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	r := &sqliteRepository{db: db}
	if err := r.createTables(); err != nil {
		return nil, err
	}
	return r, nil
}

// Close terminates the SQLite DB connection.
func (r *sqliteRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// createTables sets up the schema if it doesn't already exist.
func (r *sqliteRepository) createTables() error {
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

// -------------------------------------------------------------------
// Credential Methods
// -------------------------------------------------------------------
func (r *sqliteRepository) UpsertCredential(cred Credential) error {
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

func (r *sqliteRepository) GetCredential(code string) (*Credential, error) {
	query := `SELECT code, username, access_group, locked_out FROM credentials WHERE code = ?`
	var c Credential
	err := r.db.QueryRow(query, code).Scan(&c.Code, &c.Username, &c.AccessGroup, &c.LockedOut)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// -------------------------------------------------------------------
// AccessTime Methods
// -------------------------------------------------------------------
func (r *sqliteRepository) UpsertAccessTime(at AccessTime) error {
	query := `
		INSERT INTO access_times (access_group, start_time, end_time)
		VALUES (?, ?, ?)
		ON CONFLICT(access_group) DO UPDATE SET
			start_time = excluded.start_time,
			end_time = excluded.end_time`
	_, err := r.db.Exec(query, at.AccessGroup, at.StartTime, at.EndTime)
	return err
}

func (r *sqliteRepository) GetAccessTime(groupID int) (*AccessTime, error) {
	query := `SELECT access_group, start_time, end_time FROM access_times WHERE access_group = ?`
	var at AccessTime
	err := r.db.QueryRow(query, groupID).Scan(&at.AccessGroup, &at.StartTime, &at.EndTime)
	if err != nil {
		return nil, err
	}
	return &at, nil
}

// -------------------------------------------------------------------
// Gate Request Logs
// -------------------------------------------------------------------
func (r *sqliteRepository) AddGateLog(logEntry GateLog) error {
	query := `INSERT INTO gate_request_log (code, time, status) VALUES (?, ?, ?)`
	_, err := r.db.Exec(query, logEntry.Code, logEntry.Time, logEntry.Status)
	return err
}

func (r *sqliteRepository) GetGateLogs() ([]GateLog, error) {
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
