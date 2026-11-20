package tracker

import (
	"sync"
	"time"
)

// Default timeouts.
const (
	DefaultIdleTimeout    = 5 * time.Minute
	DefaultSessionTimeout = 30 * time.Minute
)

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
	mu             sync.Mutex
	idleTimeout    time.Duration
	sessionTimeout time.Duration
	timeMode       TimeMode
	projectStart   int64
	fileStart      int64
	lastActivity   time.Time
	currentFile    string
	isIdle         bool
}

// NewSessionTracker initializes a tracker with specified idle timeout and time mode.
func NewSessionTracker(idleTimeout time.Duration, mode TimeMode) *SessionTracker {
	if idleTimeout <= 0 {
		idleTimeout = DefaultIdleTimeout
	}

	now := time.Now()
	nowSec := now.Unix()

	return &SessionTracker{
		idleTimeout:    idleTimeout,
		sessionTimeout: DefaultSessionTimeout,
		timeMode:       mode,
		projectStart:   nowSec,
		fileStart:      nowSec,
		lastActivity:   now,
		isIdle:         false,
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

// CheckSessionExpiry resets the session start timer if elapsed inactivity exceeds the threshold.
// If resetOnReturn is true, the threshold is idleTimeout; otherwise it is sessionTimeout.
// Returns true if the session expired and a new start timestamp was recorded.
func (t *SessionTracker) CheckSessionExpiry(now time.Time, resetOnReturn bool) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	gap := t.sessionTimeout
	if resetOnReturn && t.idleTimeout > 0 {
		gap = t.idleTimeout
	}
	if gap <= 0 {
		gap = DefaultSessionTimeout
	}

	if now.Sub(t.lastActivity) >= gap {
		nowSec := now.Unix()
		t.projectStart = nowSec
		t.fileStart = nowSec
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

// SetIdleTimeout updates the inactivity duration threshold.
func (t *SessionTracker) SetIdleTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if timeout > 0 {
		t.idleTimeout = timeout
	}
}

// SetSessionTimeout updates the session gap threshold.
func (t *SessionTracker) SetSessionTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if timeout > 0 {
		t.sessionTimeout = timeout
	}
}

// ResetProject resets the project timer (e.g. when switching projects/workspaces).
func (t *SessionTracker) ResetProject() {
	t.mu.Lock()
	defer t.mu.Unlock()

	nowSec := time.Now().Unix()
	t.projectStart = nowSec
	t.fileStart = nowSec
}

// SetProjectStart overrides the project start epoch timestamp (e.g. from persisted storage).
func (t *SessionTracker) SetProjectStart(start int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if start > 0 {
		t.projectStart = start
	}
}

// SetLastActivity overrides the last activity time (e.g. from persisted storage).
func (t *SessionTracker) SetLastActivity(lastSec int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if lastSec > 0 {
		t.lastActivity = time.Unix(lastSec, 0)
	}
}
