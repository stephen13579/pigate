package gate

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	rpio "github.com/stianeikeland/go-rpio/v4"
)

const (
	pinD0        = 17 // BCM GPIO pin for Wiegand Data0
	pinD1        = 18 // BCM GPIO pin for Wiegand Data1
	frameTimeout = 25 * time.Millisecond
	pollInterval = 50 * time.Microsecond
)

// KeypadReader reads raw Wiegand pulses on D0/D1 and assembles 26‑bit codes.
type KeypadReader struct {
	d0, d1     rpio.Pin
	mu         sync.Mutex
	bits       []int
	frameTimer *time.Timer
	running    bool
}

// NewKeypadReader prepares a reader but does not start it.
func NewKeypadReader() *KeypadReader {
	return &KeypadReader{
		d0:         rpio.Pin(pinD0),
		d1:         rpio.Pin(pinD1),
		frameTimer: time.NewTimer(frameTimeout),
	}
}

// Start opens /dev/gpiomem, configures pins, and begins polling.
// onCodeReceived is called with the decimal payload when a frame completes.
func (k *KeypadReader) Start(onCodeReceived func(code string)) error {
	if err := rpio.Open(); err != nil {
		return fmt.Errorf("rpio.Open: %w", err)
	}

	// configure pins
	for _, p := range []rpio.Pin{k.d0, k.d1} {
		p.Input()
		p.PullUp()
		p.Detect(rpio.FallEdge) // enable falling‑edge detection
	}

	// drain timer so Reset works
	if !k.frameTimer.Stop() {
		<-k.frameTimer.C
	}

	k.running = true
	go k.loop(onCodeReceived)
	return nil
}

func (k *KeypadReader) loop(onCodeReceived func(code string)) {
	for k.running {
		// poll both lines
		if k.d0.EdgeDetected() {
			k.pushBit(0, onCodeReceived)
		}
		if k.d1.EdgeDetected() {
			k.pushBit(1, onCodeReceived)
		}
		time.Sleep(pollInterval)
	}
}

// pushBit appends a bit, resets the frame timer, and when timer fires
// (i.e. no new bits for frameTimeout), parses the 26‑bit frame.
func (k *KeypadReader) pushBit(bit int, onCodeReceived func(string)) {
	k.mu.Lock()
	k.bits = append(k.bits, bit)
	// reset the timer
	if !k.frameTimer.Stop() {
		<-k.frameTimer.C
	}
	k.frameTimer.Reset(frameTimeout)
	currentLen := len(k.bits)
	k.mu.Unlock()

	// launch a one‑off waiter to detect end of frame
	go func(expectLen int) {
		<-k.frameTimer.C

		k.mu.Lock()
		defer k.mu.Unlock()
		if expectLen != len(k.bits) {
			// new bits arrived since we set up this waiter
			return
		}
		raw := append([]int(nil), k.bits...)
		k.bits = nil

		code, err := parseWiegand26(raw)
		if err != nil {
			log.Println("Wiegand parse error:", err)
			return
		}
		onCodeReceived(code)
	}(currentLen)
}

// parseWiegand26 strips the two parity bits out of a 26‑bit slice and returns
// the 24‑bit payload as a decimal string.
func parseWiegand26(bits []int) (string, error) {
	if len(bits) != 26 {
		return "", errors.New("unexpected bit count")
	}
	data := bits[1 : len(bits)-1]
	var val uint32
	for _, b := range data {
		val = (val << 1) | uint32(b)
	}
	return fmt.Sprintf("%d", val), nil
}

// Stop halts polling and releases GPIO memory.
func (k *KeypadReader) Stop() {
	k.running = false
	k.d0.Detect(rpio.NoEdge)
	k.d1.Detect(rpio.NoEdge)
	rpio.Close()
}
