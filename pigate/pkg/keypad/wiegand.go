package keypad

import (
	"log"
)

type KeypadReader struct {
	// Fields for Wiegand communication
}

func NewKeypadReader() *KeypadReader {
	return &KeypadReader{}
}

func (k *KeypadReader) Start(onCodeReceived func(code string)) {
	// Initialize Wiegand reader
	// For demonstration, simulate keypad input
	for {
		code := "12345" // Simulated code
		log.Printf("Keypad input received: %s", code)
		onCodeReceived(code)
		// Sleep or wait for real input
		// time.Sleep(1 * time.Second)
	}
}
