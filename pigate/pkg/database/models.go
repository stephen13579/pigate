package database

import (
	"database/sql"
	"time"
)

type Credential struct {
	Code        string
	Username    string
	AccessGroup int
	LockedOut   bool
}

type AccessTime struct {
	AccessGroup int
	StartTime   int
	EndTime     int
}

type GateRequestLog struct {
	Code   string
	Time   time.Time
	Status string
}

func timeToMinutes(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}

func isWithinRange(current, start, end int) bool {
	if start <= end {
		return current >= start && current <= end
	}
	// Handle overnight spans
	return current >= start || current <= end
}

func ValidateCredential(db *sql.DB, code string, time time.Time) (bool, error) {
	credential, err := getCredential(db, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	accessTime, err := getAccessTime(db, credential.AccessGroup)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	// Get current time as minutes from start of day
	currentMinutes := timeToMinutes(time)

	if !isWithinRange(currentMinutes, accessTime.StartTime, accessTime.EndTime) {
		return false, nil
	}

	// Valid credential
	return true, nil
}

func getCredential(db *sql.DB, code string) (*Credential, error) {
	query := `SELECT code, username, access_group, locked_out FROM credentials WHERE code = ?`
	var credential Credential
	err := db.QueryRow(query, code).Scan(
		&credential.Code,
		&credential.Username,
		&credential.AccessGroup,
		&credential.LockedOut,
	)
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func getAccessTime(db *sql.DB, groupID int) (*AccessTime, error) {
	query := `SELECT access_group, start_time, end_time FROM access_times WHERE access_group = ?`
	var at AccessTime
	err := db.QueryRow(query, groupID).Scan(&at.AccessGroup, &at.StartTime, &at.EndTime)
	if err != nil {
		return nil, err
	}
	return &at, nil
}

func AddCredential(db *sql.DB, credential Credential) error {
	query := `INSERT INTO credentials (code, username, access_group, locked_out) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, credential.Code, credential.Username, credential.AccessGroup, credential.LockedOut)
	return err
}

func AddAccessTime(db *sql.DB, at AccessTime) error {
	query := `INSERT INTO access_times (access_group, start_time, end_time) VALUES (?, ?, ?)`
	_, err := db.Exec(query, at.AccessGroup, at.StartTime, at.EndTime)
	return err
}

func AddGateRequestLog(db *sql.DB, log GateRequestLog) error {
	query := `INSERT INTO gate_request_log (code, time, status) VALUES (?, ?, ?)`
	_, err := db.Exec(query, log.Code, log.Time, log.Status)
	return err
}

func GetGateRequestLogs(db *sql.DB) ([]GateRequestLog, error) {
	query := `SELECT code, time, status FROM gate_request_log`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	var logs []GateRequestLog
	for rows.Next() {
		var log GateRequestLog
		err := rows.Scan(&log.Code, &log.Time, &log.Status)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}
