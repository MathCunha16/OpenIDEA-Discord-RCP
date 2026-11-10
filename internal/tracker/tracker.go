package tracker

import (
	"sync"
	"time"
)

// DefaultIdleTimeout is 5 minutes.
const DefaultIdleTimeout = 5 * time.Minute

// TimeMode defines timestamp behavior for Discord Rich Presence.
type TimeMode int

const (
	// TimeModeProject counts elapsed time since the current project was opened.
	TimeModeProject TimeMode = iota
	// TimeModeFile resets the elapsed timer every time a new file is activated.
	TimeModeFile
	// TimeModeNone disables timestamps completely.
	TimeModeNone
)

// SessionTracker coordinates session start times and idle / AFK states.
type SessionTracker struct {
	mu           sync.Mutex
	idleTimeout  time.Duration
	timeMode     TimeMode
	projectStart int64
	fileStart    int64
	lastActivity time.Time
	currentFile  string
	isIdle       bool
}

// NewSessionTracker initializes a tracker with specified idle timeout and time mode.
func NewSessionTracker(idleTimeout time.Duration, mode TimeMode) *SessionTracker {
	if idleTimeout <= 0 {
		idleTimeout = DefaultIdleTimeout
	}

	now := time.Now()
	nowSec := now.Unix()

	return &SessionTracker{
		idleTimeout:  idleTimeout,
		timeMode:     mode,
		projectStart: nowSec,
		fileStart:    nowSec,
		lastActivity: now,
		isIdle:       false,
	}
}

// OnActivity records an incoming editor event, updating file and activity timestamps.
// Returns true if the tracker transitioned from idle to active.
func (t *SessionTracker) OnActivity(filePath string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	wasIdle := t.isIdle

	t.lastActivity = now
	t.isIdle = false

	if filePath != "" && filePath != t.currentFile {
		t.currentFile = filePath
		t.fileStart = now.Unix()
	}

	return wasIdle
}

// CheckIdle evaluates whether elapsed inactivity exceeds the idle timeout.
// Returns true if the state transitioned to idle.
func (t *SessionTracker) CheckIdle(now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isIdle {
		return false
	}

	if now.Sub(t.lastActivity) >= t.idleTimeout {
		t.isIdle = true
		return true
	}

	return false
}

// IsIdle returns the current idle state.
func (t *SessionTracker) IsIdle() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.isIdle
}

// SetIdle forces the tracker into idle state.
func (t *SessionTracker) SetIdle(idle bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.isIdle = idle
}

// TimestampStart returns the appropriate epoch start timestamp according to the configured TimeMode.
// Returns 0 if TimeModeNone or when idle.
func (t *SessionTracker) TimestampStart() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch t.timeMode {
	case TimeModeProject:
		return t.projectStart
	case TimeModeFile:
		return t.fileStart
	default:
		return 0
	}
}

// SetTimeMode updates the timestamp tracking strategy.
func (t *SessionTracker) SetTimeMode(mode TimeMode) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timeMode = mode
}

// ResetProject resets the project timer (e.g. when switching projects/workspaces).
func (t *SessionTracker) ResetProject() {
	t.mu.Lock()
	defer t.mu.Unlock()

	nowSec := time.Now().Unix()
	t.projectStart = nowSec
	t.fileStart = nowSec
}
