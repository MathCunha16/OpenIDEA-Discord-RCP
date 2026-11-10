package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"OpenIDEA-Discord-Rcp/internal/tracker"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if s.ApplicationID != DefaultApplicationID {
		t.Errorf("expected AppID %s, got %s", DefaultApplicationID, s.ApplicationID)
	}
	if s.ParsedTimeMode() != tracker.TimeModeProject {
		t.Errorf("expected TimeModeProject")
	}
	if s.ParsedDebounce() != 1500*time.Millisecond {
		t.Errorf("expected 1500ms debounce, got %v", s.ParsedDebounce())
	}
	if s.ParsedIdleTimeout() != 5*time.Minute {
		t.Errorf("expected 5min idle timeout, got %v", s.ParsedIdleTimeout())
	}
}

func TestLoadSettingsFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgPath := filepath.Join(tempDir, "config.json")
	content := `{
		"theme": "catppuccin_mocha",
		"time_mode": "file",
		"debounce_ms": 2000,
		"idle_timeout_mins": 10,
		"privacy_mode": true
	}`

	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loaded := LoadSettings(cfgPath)
	if loaded.Theme != "catppuccin_mocha" {
		t.Errorf("expected theme catppuccin_mocha, got %s", loaded.Theme)
	}
	if loaded.ParsedTimeMode() != tracker.TimeModeFile {
		t.Errorf("expected TimeModeFile")
	}
	if loaded.ParsedDebounce() != 2000*time.Millisecond {
		t.Errorf("expected 2000ms debounce, got %v", loaded.ParsedDebounce())
	}
	if loaded.ParsedIdleTimeout() != 10*time.Minute {
		t.Errorf("expected 10min idle timeout, got %v", loaded.ParsedIdleTimeout())
	}
	if !loaded.PrivacyMode {
		t.Errorf("expected privacy mode true")
	}
}
