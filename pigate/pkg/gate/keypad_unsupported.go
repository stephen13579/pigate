//go:build !linux

package gate

import "errors"

type KeypadReader struct{}

func NewKeypadReader() *KeypadReader {
	return &KeypadReader{}
}

func (k *KeypadReader) Start(onCodeReceived func(code string)) error {
	return errors.New("keypad reader is only supported on linux")
}

func (k *KeypadReader) Stop() {}
