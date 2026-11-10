package config

import (
	"encoding/json"
	"os"
	"time"

	"OpenIDEA-Discord-Rcp/internal/tracker"
)

// Default settings values.
const (
	DefaultApplicationID = "1569761142878896160"
	DefaultTheme         = "material"
	DefaultDebounceMs    = 1500
	DefaultIdleMinutes   = 5
)

// Settings represents the configurable preferences for OpenIDEA Discord Rich Presence.
type Settings struct {
	ApplicationID   string `json:"application_id"`
	Theme           string `json:"theme"`
	DebounceMs      int    `json:"debounce_ms"`
	IdleTimeoutMins int    `json:"idle_timeout_mins"`
	TimeMode        string `json:"time_mode"` // "project", "file", "none"
	ShowGitButton   bool   `json:"show_git_button"`
	ShowIDEButton   bool   `json:"show_ide_button"`
	PrivacyMode     bool   `json:"privacy_mode"`
}

// DefaultSettings returns the standard configuration for the daemon.
func DefaultSettings() Settings {
	return Settings{
		ApplicationID:   DefaultApplicationID,
		Theme:           DefaultTheme,
		DebounceMs:      DefaultDebounceMs,
		IdleTimeoutMins: DefaultIdleMinutes,
		TimeMode:        "project",
		ShowGitButton:   true,
		ShowIDEButton:   true,
		PrivacyMode:     false,
	}
}

// LoadSettings attempts to load settings from a JSON file, falling back to defaults.
func LoadSettings(path string) Settings {
	s := DefaultSettings()
	if path == "" {
		return s
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}

	_ = json.Unmarshal(data, &s)
	return s
}

// ParsedTimeMode maps string configuration to tracker.TimeMode.
func (s Settings) ParsedTimeMode() tracker.TimeMode {
	switch s.TimeMode {
	case "file":
		return tracker.TimeModeFile
	case "none":
		return tracker.TimeModeNone
	default:
		return tracker.TimeModeProject
	}
}

// ParsedDebounce returns the debouncing duration.
func (s Settings) ParsedDebounce() time.Duration {
	if s.DebounceMs <= 0 {
		return 1500 * time.Millisecond
	}
	return time.Duration(s.DebounceMs) * time.Millisecond
}

// ParsedIdleTimeout returns the idle duration threshold.
func (s Settings) ParsedIdleTimeout() time.Duration {
	if s.IdleTimeoutMins <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(s.IdleTimeoutMins) * time.Minute
}
