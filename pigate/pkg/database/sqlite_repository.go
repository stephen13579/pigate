package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// -------------------------------------------------------------------
// Sqlite3 Database
// -------------------------------------------------------------------
type sqliteGateManager struct {
	DB *sql.DB
	AccessManager
	AccessLogger
}

// NewRepository opens the database at dbPath, creates the required tables,
// and initializes both the AccessManager and AccessLogger.
func NewSqliteGateManager(dbPath string) (GateManager, error) {
	// Open the SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create the AccessManager (which creates its tables)
	accessMgr, err := NewSQLiteAccessManager(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Create the AccessLogger (which creates its tables)
	accessLogger, err := NewAccessLogger(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &sqliteGateManager{
		DB:            db,
		AccessManager: accessMgr,
		AccessLogger:  accessLogger,
	}, nil
}

// Close closes the underlying database connection.
func (r *sqliteGateManager) Close() error {
	return r.DB.Close()
}

// -------------------------------------------------------------------
// AccessManager
// -------------------------------------------------------------------

// Ensure sqliteRepository implements AccessManager
var _ AccessManager = (*sqlitAccessManager)(nil)

type sqlitAccessManager struct {
	db *sql.DB
}

func NewSQLiteAccessManager(db *sql.DB) (AccessManager, error) {
	manager := &sqlitAccessManager{db: db}
	if err := manager.createTables(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (r *sqlitAccessManager) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS credentials (
			code TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			access_group INTEGER NOT NULL,
			locked_out BOOLEAN NOT NULL,
			auto_update BOOLEAN NOT NULL DEFAULT 0,
			open_mode TEXT NOT NULL CHECK (open_mode IN ('regular_open', 'lock_open'))
		);`,

		`CREATE TABLE IF NOT EXISTS access_times (
			access_group INTEGER PRIMARY KEY,
			start_time TEXT NOT NULL,        -- store as local time format: "15:04:05"
			end_time TEXT NOT NULL,
			start_weekday INTEGER NOT NULL,  -- 0 = Sunday
			end_weekday INTEGER NOT NULL		
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
		INSERT INTO credentials (code, username, access_group, locked_out, auto_update, open_mode)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET
			username = excluded.username,
			access_group = excluded.access_group,
			locked_out = excluded.locked_out,
			auto_update = excluded.auto_update,
			open_mode = excluded.open_mode`
	_, err := r.db.ExecContext(ctx, query, cred.Code, cred.Username, cred.AccessGroup, cred.LockedOut, cred.AutoUpdate, cred.OpenMode)
	return err
}

// PutCredentials inserts multiple credentials into the database.
func (r *sqlitAccessManager) PutCredentials(ctx context.Context, creds []Credential) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO credentials (code, username, access_group, locked_out, auto_update, open_mode)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET
			username = excluded.username,
			access_group = excluded.access_group,
			locked_out = excluded.locked_out,
			auto_update = excluded.auto_update,
			open_mode = excluded.open_mode`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, cred := range creds {
		_, err := stmt.Exec(cred.Code, cred.Username, cred.AccessGroup, cred.LockedOut, cred.AutoUpdate, cred.OpenMode)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *sqlitAccessManager) GetCredential(ctx context.Context, code string) (*Credential, error) {
	query := `SELECT code, username, access_group, locked_out, auto_update, open_mode FROM credentials WHERE code = ?`
	var c Credential
	err := r.db.QueryRow(query, code).Scan(&c.Code, &c.Username, &c.AccessGroup, &c.LockedOut, &c.AutoUpdate, &c.OpenMode)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *sqlitAccessManager) GetCredentials(ctx context.Context) ([]Credential, error) {
	query := `SELECT code, username, access_group, locked_out, auto_update, open_mode FROM credentials`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []Credential
	for rows.Next() {
		var cred Credential
		err := rows.Scan(&cred.Code, &cred.Username, &cred.AccessGroup, &cred.LockedOut, &cred.AutoUpdate, &cred.OpenMode)
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
	query := `DELETE FROM credentials WHERE code = ?`
	_, err := r.db.ExecContext(ctx, query, code)
	if err != nil {
		return err
	}
	return nil
}

func (r *sqlitAccessManager) DeleteCredentials(ctx context.Context, codes []string) error {
	if len(codes) == 0 {
		return nil // Nothing to delete
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`DELETE FROM credentials WHERE code = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, code := range codes {
		if _, err := stmt.Exec(code); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *sqlitAccessManager) PutAccessTime(ctx context.Context, at AccessTime) error {
	query := `
		INSERT INTO access_times (
			access_group, start_time, end_time, start_weekday, end_weekday
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(access_group) DO UPDATE SET
			start_time = excluded.start_time,
			end_time = excluded.end_time,
			start_weekday = excluded.start_weekday,
			end_weekday = excluded.end_weekday`

	// Store only time-of-day as HH:MM:SS in local time
	start := at.StartTime.Format("15:04:05")
	end := at.EndTime.Format("15:04:05")

	_, err := r.db.Exec(query, at.AccessGroup, start, end, at.StartWeekday, at.EndWeekday)
	return err
}

func (r *sqlitAccessManager) GetAccessTime(ctx context.Context, groupID int) (*AccessTime, error) {
	query := `SELECT access_group, start_time, end_time, start_weekday, end_weekday FROM access_times WHERE access_group = ?`

	var (
		accessGroup  int
		startStr     string
		endStr       string
		startWeekday int
		endWeekday   int
	)

	err := r.db.QueryRowContext(ctx, query, groupID).Scan(&accessGroup, &startStr, &endStr, &startWeekday, &endWeekday)
	if err != nil {
		return nil, err
	}

	now := time.Now().Local()
	startParsed, err := time.ParseInLocation("15:04:05", startStr, now.Location())
	if err != nil {
		return nil, fmt.Errorf("failed to parse start_time: %w", err)
	}
	endParsed, err := time.ParseInLocation("15:04:05", endStr, now.Location())
	if err != nil {
		return nil, fmt.Errorf("failed to parse end_time: %w", err)
	}

	return &AccessTime{
		AccessGroup:  accessGroup,
		StartTime:    startParsed,
		EndTime:      endParsed,
		StartWeekday: time.Weekday(startWeekday),
		EndWeekday:   time.Weekday(endWeekday),
	}, nil
}

func (r *sqlitAccessManager) DeleteAccessTime(ctx context.Context, groupID int) error {
	query := `DELETE FROM access_times WHERE access_group = ?`
	_, err := r.db.ExecContext(ctx, query, groupID)
	if err != nil {
		return err
	}
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
			time INTEGER NOT NULL, -- Unix timestamp for search support
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
	_, err := r.db.Exec(query, logEntry.Code, logEntry.Time.Unix(), logEntry.Status)
	return err
}

func (r *sqliteAccessLogger) GetGateLogs(ctx context.Context) ([]GateLog, error) {
	query := `SELECT code, time, status FROM gate_request_log`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []GateLog
	for rows.Next() {
		var code string
		var ts int64
		var status GateStatus

		if err := rows.Scan(&code, &ts, &status); err != nil {
			return nil, err
		}

		logs = append(logs, GateLog{
			Code:   code,
			Time:   time.Unix(ts, 0).UTC(), // TODO: ensure this is in UTC
			Status: status,
		})
	}
	return logs, nil
}
