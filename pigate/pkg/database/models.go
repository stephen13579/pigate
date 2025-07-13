package database

import (
	"time"
)

type GateStatus string

const (
	StatusGranted GateStatus = "GRANTED"
	StatusDenied  GateStatus = "DENIED"
	StatusError   GateStatus = "ERROR"
)

type Credential struct {
	Code        string // Primary key
	Username    string
	AccessGroup int
	LockedOut   bool
	AutoUpdate  bool // “true” = this record comes from the external feed
}

type AccessTime struct {
	AccessGroup  int // Primary key // 0 - default access group
	StartTime    time.Time
	EndTime      time.Time
	StartWeekday time.Weekday
	EndWeekday   time.Weekday
}

type GateLog struct {
	Code   string // Primary key
	Time   time.Time
	Status GateStatus
}
