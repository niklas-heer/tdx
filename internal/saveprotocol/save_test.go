package saveprotocol

import (
	"errors"
	"testing"
	"time"
)

func TestFailureBoundariesAndContinuation(t *testing.T) {
	phases := []Phase{Prepare, Lock, Validate, CaptureBefore, Replace, SyncDirectory, CaptureAfter, Unlock}
	for index, failed := range phases {
		t.Run(string(failed), func(t *testing.T) {
			s := New(true, 2*time.Second)
			sentinel := errors.New("injected fault")
			var visited []Phase
			for steps := 0; s.Phase() != Done && steps < 12; steps++ {
				phase := s.Phase()
				visited = append(visited, phase)
				var err error
				if phase == failed {
					err = sentinel
				}
				s.Advance(err, 0)
			}
			out := s.Outcome()
			if s.Phase() != Done || !errors.Is(out.Err, sentinel) {
				t.Fatalf("outcome: %+v, visited %v", out, visited)
			}
			if out.Committed != (index > 4) || out.Durable != (index > 5) {
				t.Fatalf("commit/durability: %+v", out)
			}
			if index >= 2 && visited[len(visited)-1] != Unlock {
				t.Fatalf("lock cleanup skipped: %v", visited)
			}
			if failed == SyncDirectory && visited[len(visited)-2] != CaptureAfter {
				t.Fatalf("history skipped after sync failure: %v", visited)
			}
		})
	}
}
func TestVirtualLockDeadlineAndCombinedErrors(t *testing.T) {
	s := New(false, 2*time.Second)
	s.Advance(nil, 10*time.Millisecond)
	s.Advance(ErrBusy, 2009*time.Millisecond)
	if s.Phase() != Lock {
		t.Fatal("timed out early")
	}
	s.Advance(ErrBusy, 2010*time.Millisecond)
	if s.Phase() != Done || !errors.Is(s.Outcome().Err, ErrBusy) {
		t.Fatal("missing timeout")
	}
	s = New(false, 2*time.Second)
	for s.Phase() != SyncDirectory {
		s.Advance(nil, 0)
	}
	syncErr, historyErr, unlockErr := errors.New("sync"), errors.New("history"), errors.New("unlock")
	for _, err := range []error{syncErr, historyErr, unlockErr} {
		s.Advance(err, 0)
	}
	for _, err := range []error{syncErr, historyErr, unlockErr} {
		if !errors.Is(s.Outcome().Err, err) {
			t.Fatal("lost failure", err)
		}
	}
	if !s.Outcome().Committed || s.Outcome().Durable || s.Phase() != Done {
		t.Fatalf("outcome %+v", s.Outcome())
	}
}
