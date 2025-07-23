package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type postgresAccessManager struct {
	db *sql.DB
}

func (r *postgresAccessManager) InitSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS credentials (
            code TEXT PRIMARY KEY,
            username TEXT NOT NULL,
            access_group INTEGER NOT NULL,
            locked_out BOOLEAN NOT NULL,
            auto_update BOOLEAN NOT NULL
        );`,
		`CREATE TABLE IF NOT EXISTS access_times (
            access_group INTEGER PRIMARY KEY,
            start_time TIME NOT NULL,
            end_time TIME NOT NULL,
            start_weekday INTEGER NOT NULL,
            end_weekday INTEGER NOT NULL
        );`,
	}
	for _, q := range queries {
		if _, err := r.db.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func NewPostgresAccessManager(ctx context.Context, connStr string) (*postgresAccessManager, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	manager := &postgresAccessManager{db: db}
	if err := manager.InitSchema(ctx); err != nil {
		return nil, err
	}
	return manager, nil
}

// PutCredential inserts or updates a credential
func (r *postgresAccessManager) PutCredential(ctx context.Context, cred Credential) error {
	query := `
        INSERT INTO credentials (code, username, access_group, locked_out, auto_update)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (code) DO UPDATE SET
            username = EXCLUDED.username,
            access_group = EXCLUDED.access_group,
            locked_out = EXCLUDED.locked_out,
            auto_update = EXCLUDED.auto_update`
	_, err := r.db.ExecContext(ctx, query, cred.Code, cred.Username, cred.AccessGroup, cred.LockedOut, cred.AutoUpdate)
	return err
}

// PutCredentials batch inserts/updates credentials
func (r *postgresAccessManager) PutCredentials(ctx context.Context, creds []Credential) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, cred := range creds {
		if err := r.PutCredential(ctx, cred); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetCredential retrieves a credential by code
func (r *postgresAccessManager) GetCredential(ctx context.Context, code string) (*Credential, error) {
	query := `SELECT code, username, access_group, locked_out, auto_update FROM credentials WHERE code = $1`
	row := r.db.QueryRowContext(ctx, query, code)
	var cred Credential
	err := row.Scan(&cred.Code, &cred.Username, &cred.AccessGroup, &cred.LockedOut, &cred.AutoUpdate)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

// GetCredentials retrieves all credentials
func (r *postgresAccessManager) GetCredentials(ctx context.Context) ([]Credential, error) {
	query := `SELECT code, username, access_group, locked_out, auto_update FROM credentials`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var creds []Credential
	for rows.Next() {
		var cred Credential
		if err := rows.Scan(&cred.Code, &cred.Username, &cred.AccessGroup, &cred.LockedOut, &cred.AutoUpdate); err != nil {
			return nil, err
		}
		creds = append(creds, cred)
	}
	return creds, nil
}

// DeleteCredential deletes a credential by code
func (r *postgresAccessManager) DeleteCredential(ctx context.Context, code string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM credentials WHERE code = $1`, code)
	return err
}

// DeleteCredentials deletes multiple credentials by their codes
func (r *postgresAccessManager) DeleteCredentials(ctx context.Context, codes []string) error {
	if len(codes) == 0 {
		return nil
	}
	query := `DELETE FROM credentials WHERE code = ANY($1)`
	_, err := r.db.ExecContext(ctx, query, pq.Array(codes))
	return err
}

// PutAccessTime inserts or updates access time for a group
func (r *postgresAccessManager) PutAccessTime(ctx context.Context, at AccessTime) error {
	query := `
        INSERT INTO access_times (access_group, start_time, end_time, start_weekday, end_weekday)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (access_group) DO UPDATE SET
            start_time = EXCLUDED.start_time,
            end_time = EXCLUDED.end_time,
            start_weekday = EXCLUDED.start_weekday,
            end_weekday = EXCLUDED.end_weekday`
	_, err := r.db.ExecContext(ctx, query, at.AccessGroup, at.StartTime, at.EndTime, int(at.StartWeekday), int(at.EndWeekday))
	return err
}

// GetAccessTime retrieves access time for a specific group
func (r *postgresAccessManager) GetAccessTime(ctx context.Context, groupID int) (*AccessTime, error) {
	query := `SELECT access_group, start_time, end_time, start_weekday, end_weekday FROM access_times WHERE access_group = $1`
	row := r.db.QueryRowContext(ctx, query, groupID)
	var at AccessTime
	var startWeekday, endWeekday int
	err := row.Scan(&at.AccessGroup, &at.StartTime, &at.EndTime, &startWeekday, &endWeekday)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	at.StartWeekday = time.Weekday(startWeekday)
	at.EndWeekday = time.Weekday(endWeekday)
	return &at, nil
}

// DeleteAccessTime deletes access time for a group
func (r *postgresAccessManager) DeleteAccessTime(ctx context.Context, groupID int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM access_times WHERE access_group = $1`, groupID)
	return err
}
