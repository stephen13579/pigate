package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	err = createTables(db)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func createTables(db *sql.DB) error {
	credentialTable := `
    CREATE TABLE IF NOT EXISTS credentials (
        code TEXT PRIMARY KEY,
        username TEXT NOT NULL,
        access_group INTEGER NOT NULL,
        locked_out BOOLEAN NOT NULL
    );
    `
	_, err := db.Exec(credentialTable)
	if err != nil {
		return err
	}

	accessTimesTable := `
    CREATE TABLE IF NOT EXISTS access_times (
        access_group INTEGER PRIMARY KEY,
        start_time INTEGER NOT NULL,
        end_time INTEGER NOT NULL
    );
    `

	_, err = db.Exec(accessTimesTable)
	if err != nil {
		return err
	}

	gateRequestLogsTable := `
	CREATE TABLE IF NOT EXISTS gate_request_log (
		id INTEGER PRIMARY KEY,
		code TEXT NOT NULL,
		time DATETIME NOT NULL,
		status TEXT NOT NULL
	);
	`
	_, err = db.Exec(gateRequestLogsTable)
	if err != nil {
		return err
	}

	return nil
}
