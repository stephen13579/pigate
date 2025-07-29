package gate

import (
	"context"
	"log"
	"sync"
	"time"

	"pigate/pkg/database"
	"pigate/pkg/messenger"

	rpio "github.com/stianeikeland/go-rpio/v4"
)

type GateState int8

const (
	Closed GateState = iota
	Open
	LockedOpen
)

type GateController struct {
	pin              *rpio.Pin // GPIO controlling the gate relay
	ledPin           *rpio.Pin // GPIO controlling the status LED
	gm               database.GateManager
	state            GateState
	gateOpenDuration int
	mu               sync.Mutex
}

// NewGateController initializes the SPI/GPIO driver and returns a GateController.
// You only need to call this once in main.
func NewGateController(gm database.GateManager, gateOpenDuration int) *GateController {
	if err := rpio.Open(); err != nil {
		log.Fatalf("failed to open rpio: %v", err)
	}
	return &GateController{
		gm:               gm,
		state:            Closed,
		gateOpenDuration: gateOpenDuration,
	}
}

// InitPinControl configures the relay pin and LED pin in one call.
// relayPinNumber is the BCM pin driving the gate relay;
// ledPinNumber is the BCM pin driving a status LED.
func (g *GateController) InitPinControl(relayPinNumber, ledPinNumber int) {
	// Relay pin
	relay := rpio.Pin(relayPinNumber)
	relay.Output()
	relay.Low() // start closed
	g.pin = &relay

	// LED pin
	led := rpio.Pin(ledPinNumber)
	led.Output()
	led.Low() // start off
	g.ledPin = &led
}

// Open triggers either a temporary open or lock-open based on credential.
func (g *GateController) Open(code string, currentTime time.Time) error {
	if !g.ValidateCredential(code, currentTime) {
		log.Printf("Invalid credential: %s", code)
		return nil
	}
	isLockCode, err := g.isLockCode(code)
	if err != nil {
		log.Printf("Error checking lock code type: %v", err)
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	if isLockCode {
		return g.lockOpen()
	}
	return g.tempOpen()
}

// tempOpen opens gate for configured duration, unless already open or locked open.
func (g *GateController) tempOpen() error {
	if g.state == LockedOpen || g.state == Open {
		return nil
	}
	g.state = Open
	// Activate relay and LED
	g.pin.High()
	if g.ledPin != nil {
		g.ledPin.High()
	}
	// Schedule auto-close
	go func() {
		time.Sleep(time.Duration(g.gateOpenDuration) * time.Second)
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.state == Open {
			g.closeLockedOrOpen()
		}
	}()
	return nil
}

// lockOpen keeps gate open indefinitely until Close.
func (g *GateController) lockOpen() error {
	if g.state == LockedOpen {
		return nil
	}
	g.state = LockedOpen
	// Activate relay and LED
	g.pin.High()
	if g.ledPin != nil {
		g.ledPin.High()
	}
	return nil
}

// Close shuts the gate (from open or locked open) and turns off LED.
func (g *GateController) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state == Open || g.state == LockedOpen {
		// Deactivate relay and LED
		g.pin.Low()
		if g.ledPin != nil {
			g.ledPin.Low()
		}
		g.state = Closed
	}
	return nil
}

// internal helper for auto-close
func (g *GateController) closeLockedOrOpen() {
	// Deactivate relay and LED
	g.pin.Low()
	if g.ledPin != nil {
		g.ledPin.Low()
	}
	g.state = Closed
}

func (g *GateController) isLockCode(code string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cred, err := g.gm.GetCredential(ctx, code)
	if err != nil {
		return false, err
	}
	return cred.OpenMode == database.LockOpen, nil
}

// ValidateCredential checks if credential is valid and within allowed time.
func (g *GateController) ValidateCredential(code string, currentTime time.Time) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cred, err := g.gm.GetCredential(ctx, code)
	if err != nil || cred.LockedOut {
		return false
	}
	at, err := g.gm.GetAccessTime(ctx, cred.AccessGroup)
	if err != nil {
		return false
	}
	return isTimeOfDayInRange(currentTime, at.StartTime, at.EndTime)
}

// isTimeOfDayInRange ignores date—only compares times, handling overnight spans.
func isTimeOfDayInRange(current, start, end time.Time) bool {
	c := time.Date(0, 1, 1, current.Hour(), current.Minute(), current.Second(), 0, time.UTC)
	s := time.Date(0, 1, 1, start.Hour(), start.Minute(), start.Second(), 0, time.UTC)
	e := time.Date(0, 1, 1, end.Hour(), end.Minute(), end.Second(), 0, time.UTC)
	if !s.After(e) {
		return (c.Equal(s) || c.After(s)) && (c.Equal(e) || c.Before(e))
	}
	return c.Equal(s) || c.After(s) || c.Before(e)
}

// CommandHandler returns a function to handle remote commands: open, close, hold open.
func (g *GateController) CommandHandler() func(topic, msg string) {
	return func(topic, msg string) {
		log.Printf("Received command on topic %s: %s", topic, msg)
		switch msg {
		case messenger.CommandOpenMessage:
			log.Println("Opening the gate...")
			_ = g.tempOpen()
		case messenger.CommandCloseMessage:
			log.Println("Closing the gate...")
			_ = g.Close()
		case messenger.CommandHoldOpenMessage:
			log.Println("Locking the gate open...")
			_ = g.lockOpen()
		default:
			log.Printf("Unknown command received: %s", msg)
		}
	}
}
