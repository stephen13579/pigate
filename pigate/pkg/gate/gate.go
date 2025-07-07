package gate

import (
	"context"
	"log"
	"sync"
	"time"

	"pigate/pkg/database"
	"pigate/pkg/messenger"

	"github.com/stianeikeland/go-rpio"
)

type GateState int8

const (
	Closed GateState = iota
	Open
	LockedOpen
)

type GateController struct {
	pin              *rpio.Pin
	repository       database.Repository
	state            GateState
	gateOpenDuration int
	mu               sync.Mutex
}

func NewGateController(repo *database.Repository, gateOpenDuration int) *GateController {
	return &GateController{
		repository:       *repo,
		state:            Closed,
		gateOpenDuration: gateOpenDuration,
	}
}

// Initialize the pin used to control the gate (high for open command, low for close command)
func (g *GateController) InitPinControl(pinNumber int) error {
	err := rpio.Open()
	if err != nil {
		return err
	}
	pin := rpio.Pin(pinNumber)
	pin.Output()
	g.pin = &pin
	return nil
}

// Open activates the gate for the specified duration in seconds
// If the gate is already open, it does not restart the timer unless its in LockedOpen state
func (g *GateController) Open() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == LockedOpen {
		// Locked open: ignore open commands
		return nil
	}

	if g.state == Open {
		// Already open: ignore open commands
		return nil
	}

	g.state = Open
	g.pin.High()

	// Start a goroutine to close the gate after the duration
	go func() {
		time.Sleep(time.Duration(g.gateOpenDuration) * time.Second)
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.state == Open {
			g.Close()
		}
	}()

	return nil
}

// LockOpen keeps the gate open indefinitely until explicitly closed (or system is restarted: default is closed state)
func (g *GateController) LockOpen() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == LockedOpen {
		// Already locked open: do nothing
		return nil
	}

	g.state = LockedOpen
	g.pin.High()

	return nil
}

// Close shuts the gate if its in LockedOpen state
func (g *GateController) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == LockedOpen {
		g.pin.Low()
		g.state = Closed
	}

	return nil
}

// ValidateCredential validates a credential based on the repository data
func (g *GateController) ValidateCredential(code string, currentTime time.Time) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	credential, err := g.repository.AccessMgr.GetCredential(ctx, code)
	if err != nil {
		return false
	}

	if credential.LockedOut {
		return false
	}

	accessTime, err := g.repository.AccessMgr.GetAccessTime(ctx, credential.AccessGroup)
	if err != nil {
		return false
	}

	return isTimeOfDayInRange(currentTime, accessTime.StartTime, accessTime.EndTime)
}

// isTimeOfDayInRange checks if the current time of day is within a time range (inclusive).
// It ignores the year, month, and day — only compares hour, minute, second.
// Handles overnight ranges (e.g., 22:00–06:00).
func isTimeOfDayInRange(current, start, end time.Time) bool {
	// Normalize all to same dummy date
	currentTOD := time.Date(0, 1, 1, current.Hour(), current.Minute(), current.Second(), 0, time.UTC)
	startTOD := time.Date(0, 1, 1, start.Hour(), start.Minute(), start.Second(), 0, time.UTC)
	endTOD := time.Date(0, 1, 1, end.Hour(), end.Minute(), end.Second(), 0, time.UTC)

	if startTOD.Before(endTOD) || startTOD.Equal(endTOD) {
		// Normal daytime range
		return (currentTOD.After(startTOD) || currentTOD.Equal(startTOD)) &&
			(currentTOD.Before(endTOD) || currentTOD.Equal(endTOD))
	}
	// Overnight range
	return currentTOD.After(startTOD) || currentTOD.Before(endTOD) ||
		currentTOD.Equal(startTOD) || currentTOD.Equal(endTOD)
}

func (g *GateController) CommandHandler() func(topic string, msg string) {
	return func(topic string, msg string) {
		log.Printf("Received command on topic %s: %s", topic, msg)

		switch msg {
		case messenger.CommandOpenMessage:
			log.Println("Opening the gate...")
			if err := g.Open(); err != nil {
				log.Printf("Failed to open gate: %v", err)
			}
		case messenger.CommandCloseMessage:
			log.Println("Closing the gate...")
			if err := g.Close(); err != nil {
				log.Printf("Failed to close gate: %v", err)
			}
		case messenger.CommandHoldOpenMessage:
			log.Println("Locking the gate open...")
			if err := g.LockOpen(); err != nil {
				log.Printf("Failed to lock gate open: %v", err)
			}
		default:
			log.Printf("Unknown command received: %s", msg)
		}
	}
}
