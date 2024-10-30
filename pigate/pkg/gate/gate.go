package gate

import (
	"time"

	"github.com/stianeikeland/go-rpio"
)

type GateController struct {
	pin rpio.Pin
}

func NewGateController(pinNumber int) (*GateController, error) {
	err := rpio.Open()
	if err != nil {
		return nil, err
	}
	pin := rpio.Pin(pinNumber)
	pin.Output()
	return &GateController{pin: pin}, nil
}

func (g *GateController) Open(duration int) {
	g.pin.High()
	time.Sleep(time.Duration(duration) * time.Second)
	g.pin.Low()
}

func (g *GateController) Close() {
	g.pin.Low()
	rpio.Close()
}
