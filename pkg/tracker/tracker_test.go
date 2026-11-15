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

func TestSessionTrackerSetProjectStart(t *testing.T) {
	tr := NewSessionTracker(time.Minute, TimeModeProject)
	customStart := int64(1234567890)
	tr.SetProjectStart(customStart)
	if tr.TimestampStart() != customStart {
		t.Fatalf("expected project start %d, got %d", customStart, tr.TimestampStart())
	}
}

func TestSessionTrackerSetLastActivity(t *testing.T) {
	timeout := 10 * time.Minute
	tr := NewSessionTracker(timeout, TimeModeProject)

	// Set last activity to 15 minutes ago
	fifteenMinsAgo := time.Now().Add(-15 * time.Minute).Unix()
	tr.SetLastActivity(fifteenMinsAgo)

	if !tr.CheckIdle(time.Now()) {
		t.Fatalf("expected tracker to transition to idle after setting lastActivity in the past")
	}
	if !tr.IsIdle() {
		t.Fatalf("expected tracker to be idle")
	}

	// A reactivation after an idle reclaim must restore the idle state from
	// the persisted timestamp, not from a fresh activity.
	tr2 := NewSessionTracker(timeout, TimeModeProject)
	tr2.SetLastActivity(fifteenMinsAgo)
	if !tr2.CheckIdle(time.Now()) {
		t.Fatalf("expected reactivated tracker to be idle")
	}

	// Real user activity must clear idle again.
	if wasIdle := tr2.OnActivity("main.go"); !wasIdle {
		t.Fatalf("expected OnActivity to report idle recovery")
	}
	if tr2.IsIdle() {
		t.Fatalf("expected tracker to be active after user activity")
	}
}
