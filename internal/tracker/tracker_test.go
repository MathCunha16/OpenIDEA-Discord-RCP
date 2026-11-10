package tracker

import (
	"testing"
	"time"
)

func TestSessionTrackerIdleTransition(t *testing.T) {
	timeout := 100 * time.Millisecond
	tr := NewSessionTracker(timeout, TimeModeProject)

	if tr.IsIdle() {
		t.Fatalf("expected tracker to start active")
	}

	// Inactivity under timeout
	now := time.Now()
	if tr.CheckIdle(now.Add(50 * time.Millisecond)) {
		t.Fatalf("expected CheckIdle to return false before timeout")
	}
	if tr.IsIdle() {
		t.Fatalf("expected tracker to remain active")
	}

	// Inactivity exceeding timeout
	if !tr.CheckIdle(now.Add(150 * time.Millisecond)) {
		t.Fatalf("expected CheckIdle to return true after timeout")
	}
	if !tr.IsIdle() {
		t.Fatalf("expected tracker to be idle")
	}

	// Subsequent check returns false (already idle)
	if tr.CheckIdle(now.Add(200 * time.Millisecond)) {
		t.Fatalf("expected CheckIdle to return false when already idle")
	}

	// Activity resumes
	resumed := tr.OnActivity("main.go")
	if !resumed {
		t.Fatalf("expected OnActivity to report recovery from idle")
	}
	if tr.IsIdle() {
		t.Fatalf("expected tracker to be active after activity")
	}
}

func TestSessionTrackerTimeModes(t *testing.T) {
	tr := NewSessionTracker(time.Minute, TimeModeProject)

	pStart := tr.TimestampStart()
	if pStart == 0 {
		t.Fatalf("expected non-zero project start timestamp")
	}

	// Switch to TimeModeFile
	tr.SetTimeMode(TimeModeFile)
	fStart := tr.TimestampStart()
	if fStart == 0 {
		t.Fatalf("expected non-zero file start timestamp")
	}

	// Switch to TimeModeNone
	tr.SetTimeMode(TimeModeNone)
	if tr.TimestampStart() != 0 {
		t.Fatalf("expected 0 timestamp for TimeModeNone")
	}
}

func TestSessionTrackerFileChange(t *testing.T) {
	tr := NewSessionTracker(time.Minute, TimeModeFile)

	tr.OnActivity("file1.go")
	initialStart := tr.TimestampStart()

	// Wait small duration
	time.Sleep(10 * time.Millisecond)

	// Activity on same file does not reset start
	tr.OnActivity("file1.go")
	if tr.TimestampStart() != initialStart {
		t.Fatalf("start timestamp should not reset for the same file")
	}
}
