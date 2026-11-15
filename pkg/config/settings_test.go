package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"OpenIDEA-Discord-Rcp/pkg/tracker"
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

func TestLoadFromGetter(t *testing.T) {
	mockSettings := map[string]string{
		"enabled":               "true",
		"detectIdle":            "false",
		"idleTimeoutMins":       "15",
		"whileIdle":             `"Hide Presence"`,
		"resetOpenTimeOnReturn": "true",
		"fileFirstLine":         `"File Name"`,
		"fileSecondLine":        `"File name only"`,
		"projectFirstLine":      `"Project Name only"`,
		"projectSecondLine":     `"In project"`,
		"prefixReadingEditing":  "false",
		"timeMode":              `"File Open Time"`,
		"applicationTheme":      `"Catppuccin Mocha"`,
		"languageTheme":         `"Material"`,
		"privacyMode":           "true",
		"showGitButton":         "false",
		"showIDEButton":         "false",
	}

	getter := func(key string) (string, bool) {
		val, ok := mockSettings[key]
		return val, ok
	}

	loaded := LoadFromGetter(getter)
	if !loaded.Enabled {
		t.Errorf("expected enabled true")
	}
	if loaded.DetectIdle {
		t.Errorf("expected detectIdle false")
	}
	if loaded.IdleTimeoutMins != 15 {
		t.Errorf("expected idleTimeoutMins 15, got %d", loaded.IdleTimeoutMins)
	}
	if loaded.WhileIdle != "hide_presence" {
		t.Errorf("expected whileIdle hide_presence, got %s", loaded.WhileIdle)
	}
	if !loaded.ResetOpenTimeOnReturn {
		t.Errorf("expected resetOpenTimeOnReturn true")
	}
	if loaded.FileFirstLine != "{file}" {
		t.Errorf("expected fileFirstLine '{file}', got %s", loaded.FileFirstLine)
	}
	if loaded.FileSecondLine != "{file}" {
		t.Errorf("expected fileSecondLine '{file}', got %s", loaded.FileSecondLine)
	}
	if loaded.ProjectFirstLine != "{project}" {
		t.Errorf("expected projectFirstLine '{project}', got %s", loaded.ProjectFirstLine)
	}
	if loaded.ProjectSecondLine != "Idle in project" {
		t.Errorf("expected projectSecondLine 'Idle in project', got %s", loaded.ProjectSecondLine)
	}
	if loaded.PrefixReadingEditing {
		t.Errorf("expected prefixReadingEditing false")
	}
	if loaded.ParsedTimeMode() != tracker.TimeModeFile {
		t.Errorf("expected ParsedTimeMode TimeModeFile")
	}
	if loaded.ApplicationTheme != "catppuccin_mocha" {
		t.Errorf("expected applicationTheme catppuccin_mocha, got %s", loaded.ApplicationTheme)
	}
	if loaded.LanguageTheme != "material" {
		t.Errorf("expected languageTheme material, got %s", loaded.LanguageTheme)
	}
	if !loaded.PrivacyMode {
		t.Errorf("expected privacyMode true")
	}
	if loaded.ShowGitButton {
		t.Errorf("expected showGitButton false")
	}
	if loaded.ShowIDEButton {
		t.Errorf("expected showIDEButton false")
	}
}
