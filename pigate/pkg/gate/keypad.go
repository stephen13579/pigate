package gate

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	gpiocdev "github.com/warthog618/go-gpiocdev"
)

const (
	pinD0 = 17 // BCM GPIO pin for Wiegand Data0
	pinD1 = 18 // BCM GPIO pin for Wiegand Data1

	keyFrameTimeout  = 100 * time.Millisecond // max gap between bits of one key
	codeFrameTimeout = 3 * time.Second        // max gap between keys in a code
	maxKeysPerCode   = 5                      // 5-key PIN
)

// KeypadReader reads raw Wiegand pulses on D0/D1 and assembles keypad codes.
type KeypadReader struct {
	d0, d1 *gpiocdev.Line

	bitCh  chan int
	stopCh chan struct{}
}

// NewKeypadReader prepares a reader but does not start it.
func NewKeypadReader() *KeypadReader {
	return &KeypadReader{
		bitCh:  make(chan int, 64),
		stopCh: make(chan struct{}),
	}
}

// Start requests GPIO lines via gpiocdev and installs edge handlers.
// onCodeReceived is called with the full 5-key code string (e.g. "12345").
func (k *KeypadReader) Start(onCodeReceived func(code string)) error {
	// D0 handler = bit 0
	d0Handler := func(evt gpiocdev.LineEvent) {
		k.enqueueBit(0)
	}

	// D1 handler = bit 1
	d1Handler := func(evt gpiocdev.LineEvent) {
		k.enqueueBit(1)
	}

	// Request D0 line: input, falling edges, edge handler
	d0, err := gpiocdev.RequestLine(
		"gpiochip0",
		pinD0,
		gpiocdev.AsInput,
		gpiocdev.WithEventHandler(d0Handler),
		gpiocdev.WithFallingEdge,
	)
	if err != nil {
		return fmt.Errorf("request D0 line: %w", err)
	}

	// Request D1 line: input, falling edges, edge handler
	d1, err := gpiocdev.RequestLine(
		"gpiochip0",
		pinD1,
		gpiocdev.AsInput,
		gpiocdev.WithEventHandler(d1Handler),
		gpiocdev.WithFallingEdge,
	)
	if err != nil {
		_ = d0.Close()
		return fmt.Errorf("request D1 line: %w", err)
	}

	k.d0 = d0
	k.d1 = d1

	log.Println("Wiegand: keypad reader started (edge-driven, 4-bit keys, 5-key codes)")

	go k.run(onCodeReceived)
	return nil
}

// enqueueBit is called from the edge handlers; it must not block
func (k *KeypadReader) enqueueBit(bit int) {
	select {
	case k.bitCh <- bit:
	default:
		log.Println("Wiegand: bit channel full, dropping bit")
	}
}

// run implements:
// 1) 4-bit key frames with <keyFrameTimeout>ms timeout between bits
// 2) Up to 5 keys per code, <codeFrameTimeout>s inactivity timeout between keys
func (k *KeypadReader) run(onCodeReceived func(code string)) {
	var keyBits []int // bits for current key (expect 4)
	var keys []string // collected keys for current code (expect up to 5)

	keyTimer := time.NewTimer(time.Hour)  // dummy long; we'll stop immediately
	codeTimer := time.NewTimer(time.Hour) // same
	if !keyTimer.Stop() {
		<-keyTimer.C
	}
	if !codeTimer.Stop() {
		<-codeTimer.C
	}

	for {
		select {
		case <-k.stopCh:
			return

		case bit := <-k.bitCh:
			// New bit received: extend current key and (re)start key timer
			keyBits = append(keyBits, bit)

			// reset key timeout
			if !keyTimer.Stop() {
				select {
				case <-keyTimer.C:
				default:
				}
			}
			keyTimer.Reset(keyFrameTimeout)

			// if key complete (4 bits), parse it
			if len(keyBits) == 4 {
				key, _, err := parseKeypad4(keyBits)
				if err != nil {
					log.Printf("Wiegand keypad parse error for bits %v: %v\n", keyBits, err)
				} else {
					keys = append(keys, key)

					// reset code timeout (we got a fresh key)
					if !codeTimer.Stop() {
						select {
						case <-codeTimer.C:
						default:
						}
					}
					codeTimer.Reset(codeFrameTimeout)

					// if we have 5 keys, send immediately
					if len(keys) == maxKeysPerCode {
						code := strings.Join(keys, "")
						onCodeReceived(code)
						keys = nil

						// stop code timer until next key
						if !codeTimer.Stop() {
							select {
							case <-codeTimer.C:
							default:
							}
						}
					}
				}

				// key is done, reset keyBits and keyTimer
				keyBits = nil
				if !keyTimer.Stop() {
					select {
					case <-keyTimer.C:
					default:
					}
				}
			}

		case <-keyTimer.C:
			// Too much gap inside a key -> discard partial key
			if len(keyBits) > 0 {
				log.Printf("Wiegand: key timeout, discarding partial bits: %v\n", keyBits)
				keyBits = nil
			}

		case <-codeTimer.C:
			// Too much gap between keys -> send whatever we have (if any)
			if len(keys) > 0 {
				code := strings.Join(keys, "")
				onCodeReceived(code)
				keys = nil
			}
		}
	}
}

// parseKeypad4 interprets a 4-bit Wiegand keypad frame.
// Typical mapping:
//
//	0x0-0x9 => '0'-'9'
//	0xA     => '*'
//	0xB     => '#'
func parseKeypad4(bits []int) (string, uint8, error) {
	if len(bits) != 4 {
		return "", 0, errors.New("expected 4-bit keypad frame")
	}

	var val uint8
	for _, b := range bits {
		if b != 0 && b != 1 {
			return "", 0, fmt.Errorf("invalid bit %d in frame", b)
		}
		val = (val << 1) | uint8(b)
	}

	var key string
	switch val {
	case 0x0:
		key = "0"
	case 0x1:
		key = "1"
	case 0x2:
		key = "2"
	case 0x3:
		key = "3"
	case 0x4:
		key = "4"
	case 0x5:
		key = "5"
	case 0x6:
		key = "6"
	case 0x7:
		key = "7"
	case 0x8:
		key = "8"
	case 0x9:
		key = "9"
	case 0xA:
		key = "*"
	case 0xB:
		key = "#"
	default:
		key = "?"
	}

	return key, val, nil
}

// Stop halts edge watching and releases GPIO lines.
func (k *KeypadReader) Stop() {
	log.Println("Wiegand: stopping keypad reader (gpiocdev)")
	close(k.stopCh)
	if k.d0 != nil {
		_ = k.d0.Close()
	}
	if k.d1 != nil {
		_ = k.d1.Close()
	}
}
