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
	if loaded.ApplicationTheme != "catppuccin_mocha" {
		t.Errorf("expected applicationTheme catppuccin_mocha, got %s", loaded.ApplicationTheme)
	}
	if loaded.LanguageTheme != "catppuccin_mocha" {
		t.Errorf("expected languageTheme catppuccin_mocha, got %s", loaded.LanguageTheme)
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

func TestUnmarshalCamelCaseSettings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-camel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgPath := filepath.Join(tempDir, "config.json")
	content := `{
		"applicationId": "9999999999",
		"theme": "atom",
		"timeMode": "none",
		"debounceMs": 3000,
		"idleTimeoutMins": 15,
		"privacyMode": true,
		"showGitButton": false,
		"showIDEButton": false
	}`

	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loaded := LoadSettings(cfgPath)
	if loaded.ApplicationID != "9999999999" {
		t.Errorf("expected applicationId 9999999999, got %s", loaded.ApplicationID)
	}
	if loaded.Theme != "atom" {
		t.Errorf("expected theme atom, got %s", loaded.Theme)
	}
	if loaded.ParsedTimeMode() != tracker.TimeModeNone {
		t.Errorf("expected TimeModeNone, got %v", loaded.ParsedTimeMode())
	}
	if loaded.ParsedDebounce() != 3000*time.Millisecond {
		t.Errorf("expected 3000ms debounce, got %v", loaded.ParsedDebounce())
	}
	if loaded.ParsedIdleTimeout() != 15*time.Minute {
		t.Errorf("expected 15min idle timeout, got %v", loaded.ParsedIdleTimeout())
	}
	if !loaded.PrivacyMode {
		t.Errorf("expected privacy mode true")
	}
	if loaded.ShowGitButton {
		t.Errorf("expected showGitButton false")
	}
	if loaded.ShowIDEButton {
		t.Errorf("expected showIDEButton false")
	}
}

func TestFullSchemaSettings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-full-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgPath := filepath.Join(tempDir, "config.json")
	content := `{
		"enabled": true,
		"detectIdle": false,
		"idleTimeoutMins": 10,
		"whileIdle": "keep_showing",
		"resetOpenTimeOnReturn": true,
		"fileFirstLine": "Custom Project: {project}",
		"fileSecondLine": "Working on {file}",
		"prefixReadingEditing": false,
		"fileLargeIconRole": "application",
		"fileSmallIconRole": "none",
		"fileTimeMode": "file",
		"projectFirstLine": "Just {project}",
		"projectSecondLine": "Idle in project",
		"projectTimeMode": "none",
		"applicationTheme": "atom",
		"languageTheme": "catppuccin_mocha",
		"privacyMode": false,
		"showGitButton": true,
		"showIDEButton": false,
		"debounceMs": 2500,
		"applicationId": "1122334455"
	}`

	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loaded := LoadSettings(cfgPath)
	if !loaded.Enabled {
		t.Errorf("expected enabled true")
	}
	if loaded.DetectIdle {
		t.Errorf("expected detectIdle false")
	}
	if loaded.WhileIdle != "keep_showing" {
		t.Errorf("expected whileIdle keep_showing, got %s", loaded.WhileIdle)
	}
	if !loaded.ResetOpenTimeOnReturn {
		t.Errorf("expected resetOpenTimeOnReturn true")
	}
	if loaded.FileFirstLine != "Custom Project: {project}" {
		t.Errorf("expected fileFirstLine 'Custom Project: {project}', got %s", loaded.FileFirstLine)
	}
	if loaded.ApplicationTheme != "atom" {
		t.Errorf("expected applicationTheme atom, got %s", loaded.ApplicationTheme)
	}
	if loaded.LanguageTheme != "catppuccin_mocha" {
		t.Errorf("expected languageTheme catppuccin_mocha, got %s", loaded.LanguageTheme)
	}
	if loaded.FileLargeIconRole != "application" {
		t.Errorf("expected fileLargeIconRole application, got %s", loaded.FileLargeIconRole)
	}
}

