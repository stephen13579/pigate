//go:build !linux

package gate

type noopOutputPin struct{}

func openPinDriver() error {
	return nil
}

func newOutputPin(pinNumber int) outputPin {
	return noopOutputPin{}
}

func (noopOutputPin) High() {}

func (noopOutputPin) Low() {}
