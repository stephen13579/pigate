package gate

import (
	"time"
)

type Timer struct {
	duration time.Duration
	timer    *time.Timer
}

func NewTimer(seconds int) *Timer {
	return &Timer{
		duration: time.Duration(seconds) * time.Second,
	}
}

func (t *Timer) Start(callback func()) {
	t.timer = time.AfterFunc(t.duration, callback)
}

func (t *Timer) Stop() {
	if t.timer != nil {
		t.timer.Stop()
	}
}
