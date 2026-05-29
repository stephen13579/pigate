//go:build linux

package gate

import rpio "github.com/stianeikeland/go-rpio/v4"

type gpioOutputPin struct {
	pin rpio.Pin
}

func openPinDriver() error {
	return rpio.Open()
}

func newOutputPin(pinNumber int) outputPin {
	pin := rpio.Pin(pinNumber)
	pin.Output()
	pin.Low()
	return &gpioOutputPin{pin: pin}
}

func (p *gpioOutputPin) High() {
	p.pin.High()
}

func (p *gpioOutputPin) Low() {
	p.pin.Low()
}
