package gate

import (
	"log"
	"sync"
	"time"

	"pigate/pkg/database"

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

func NewGateController(repo database.Repository, gateOpenDuration int) *GateController {
	return &GateController{
		repository:       repo,
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
	credential, err := g.repository.GetCredential(code)
	if err != nil {
		return false
	}

	if credential.LockedOut {
		return false
	}

	accessTime, err := g.repository.GetAccessTime(credential.AccessGroup)
	if err != nil {
		return false
	}

	// Convert current time to minutes since the start of the day
	currentMinutes := timeToMinutes(currentTime)

	return isWithinRange(currentMinutes, accessTime.StartTime, accessTime.EndTime)
}

// timeToMinutes converts a time.Time object to minutes since the start of the day
func timeToMinutes(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}

// isWithinRange checks if a given time in minutes is within a start and end range
func isWithinRange(current, start, end int) bool {
	if start <= end {
		return current >= start && current <= end
	}
	// Handle overnight spans (e.g., 10 PM to 6 AM)
	return current >= start || current <= end
}

func (g *GateController) CommandHandler() func(topic string, msg string) {
	return func(topic string, msg string) {
		log.Printf("Received command on topic %s: %s", topic, msg)

		switch msg {
		case "OPEN":
			log.Println("Opening the gate...")
			if err := g.Open(); err != nil {
				log.Printf("Failed to open gate: %v", err)
			}
		case "CLOSE":
			log.Println("Closing the gate...")
			if err := g.Close(); err != nil {
				log.Printf("Failed to close gate: %v", err)
			}
		case "LOCK_OPEN":
			log.Println("Locking the gate open...")
			if err := g.LockOpen(); err != nil {
				log.Printf("Failed to lock gate open: %v", err)
			}
		default:
			log.Printf("Unknown command received: %s", msg)
		}
	}
}
