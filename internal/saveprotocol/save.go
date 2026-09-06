// Package saveprotocol orders the effects of both native and simulated saves.
// It performs no I/O and obtains monotonic elapsed time from its driver.
package saveprotocol

import (
	"errors"
	"fmt"
	"time"
)

type Phase string

const (
	Prepare       Phase = "Prepare"
	Lock          Phase = "Lock"
	Validate      Phase = "Validate"
	CaptureBefore Phase = "CaptureBefore"
	Replace       Phase = "Replace"
	SyncDirectory Phase = "SyncDirectory"
	CaptureAfter  Phase = "CaptureAfter"
	Unlock        Phase = "Unlock"
	Done          Phase = "Done"
)

var ErrBusy = errors.New("file is busy in another tdx process")

type Outcome struct {
	Committed bool
	Durable   bool
	Err       error
}
type Save struct {
	phase                Phase
	force, locked        bool
	lockStarted, timeout time.Duration
	outcome              Outcome
}

func New(force bool, timeout time.Duration) *Save {
	return &Save{phase: Prepare, force: force, timeout: timeout}
}
func (s *Save) Phase() Phase     { return s.phase }
func (s *Save) Outcome() Outcome { return s.outcome }

// Advance completes one pending effect. ErrBusy retries only lock acquisition,
// and only within the driver-supplied monotonic time budget.
func (s *Save) Advance(err error, now time.Duration) {
	if s.phase == Done {
		return
	}
	if errors.Is(err, ErrBusy) && s.phase == Lock && now-s.lockStarted < s.timeout {
		return
	}
	if err != nil {
		if s.phase == SyncDirectory || s.phase == CaptureAfter || s.phase == Unlock {
			label := map[Phase]string{SyncDirectory: "sync parent directory", CaptureAfter: "capture saved version", Unlock: "unlock file"}[s.phase]
			err = fmt.Errorf("%s: %w", label, err)
		}
		s.outcome.Err = errors.Join(s.outcome.Err, err)
		switch s.phase {
		case SyncDirectory:
			s.phase = CaptureAfter
		case CaptureAfter:
			s.phase = Unlock
		case Unlock:
			s.locked = false
			s.phase = Done
		default:
			if s.locked {
				s.phase = Unlock
			} else {
				s.phase = Done
			}
		}
		return
	}
	switch s.phase {
	case Prepare:
		s.lockStarted = now
		s.phase = Lock
	case Lock:
		s.locked = true
		s.phase = Validate
	case Validate:
		if s.force {
			s.phase = CaptureBefore
		} else {
			s.phase = Replace
		}
	case CaptureBefore:
		s.phase = Replace
	case Replace:
		s.outcome.Committed = true
		s.phase = SyncDirectory
	case SyncDirectory:
		s.outcome.Durable = true
		s.phase = CaptureAfter
	case CaptureAfter:
		s.phase = Unlock
	case Unlock:
		s.locked = false
		s.phase = Done
	}
}
