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
	// SQL statements to create tables
	credentialTable := `
    CREATE TABLE IF NOT EXISTS credentials (
        id INTEGER PRIMARY KEY,
        code TEXT UNIQUE,
        group_id INTEGER,
        valid_from DATETIME,
        valid_to DATETIME
    );
    `
	groupTable := `
    CREATE TABLE IF NOT EXISTS groups (
        id INTEGER PRIMARY KEY,
        name TEXT,
        access_start_time TIME,
        access_end_time TIME
    );
    `
	_, err := db.Exec(credentialTable)
	if err != nil {
		return err
	}
	_, err = db.Exec(groupTable)
	return err
}
