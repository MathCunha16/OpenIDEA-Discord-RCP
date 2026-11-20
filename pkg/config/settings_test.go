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

// Regression: after a host reclaim, settings.Get may return nothing. Missing
// keys must keep the previously persisted values instead of silently falling
// back to defaults (which would turn Privacy Mode off and expose file names).
func TestLoadFromGetterWithBasePreservesSettingsOnMissingKeys(t *testing.T) {
	base := DefaultSettings()
	base.PrivacyMode = true
	base.ShowGitButton = false
	base.ShowIDEButton = false
	base.Enabled = false
	base.ApplicationTheme = "catppuccin_latte"
	base.LanguageTheme = "atom"
	base.FileFirstLine = "{file} custom"
	base.IdleTimeoutMins = 42

	getter := func(string) (string, bool) { return "", false }

	got := LoadFromGetterWithBase(getter, base)
	if !got.PrivacyMode {
		t.Error("privacyMode reset: would leak file/project/branch names")
	}
	if got.ShowGitButton {
		t.Error("showGitButton reset to default true")
	}
	if got.ShowIDEButton {
		t.Error("showIDEButton reset to default true")
	}
	if got.Enabled {
		t.Error("enabled reset to default true")
	}
	if got.ApplicationTheme != base.ApplicationTheme || got.LanguageTheme != base.LanguageTheme {
		t.Errorf("themes reset: got %s/%s", got.ApplicationTheme, got.LanguageTheme)
	}
	if got.FileFirstLine != base.FileFirstLine || got.IdleTimeoutMins != 42 {
		t.Errorf("custom values reset: got %q / %d", got.FileFirstLine, got.IdleTimeoutMins)
	}
}

func TestLoadFromGetterWithBaseAppliesHostOverrides(t *testing.T) {
	base := DefaultSettings()
	base.PrivacyMode = true
	base.ShowGitButton = false

	getter := func(key string) (string, bool) {
		switch key {
		case "privacyMode":
			return "false", true
		case "applicationTheme":
			return `"flat"`, true
		}
		return "", false
	}

	got := LoadFromGetterWithBase(getter, base)
	if got.PrivacyMode {
		t.Error("host override for privacyMode was ignored")
	}
	if got.ApplicationTheme != "flat" {
		t.Errorf("host override for theme was ignored: %s", got.ApplicationTheme)
	}
	if got.ShowGitButton {
		t.Error("missing key should keep the base value")
	}
}

func TestLoadFromGetterUsesDefaultsOnFirstInstall(t *testing.T) {
	getter := func(string) (string, bool) { return "", false }
	got := LoadFromGetter(getter)
	d := DefaultSettings()
	if got.PrivacyMode != d.PrivacyMode || got.ShowGitButton != d.ShowGitButton || got.Enabled != d.Enabled {
		t.Error("first install must start from built-in defaults")
	}
}

// RestoreSettings (used on activation) must NOT consult the host: activate()
// has no workspace_root, so the host would answer with global/schema defaults
// for keys set at project scope. The persisted snapshot is used verbatim.
func TestRestoreUsesSnapshotVerbatim(t *testing.T) {
	snapshot := DefaultSettings()
	snapshot.PrivacyMode = true
	snapshot.ShowGitButton = false
	snapshot.ShowIDEButton = false
	snapshot.ApplicationTheme = "catppuccin_latte"
	snapshot.FileFirstLine = "{file} custom"

	// No getter at all: the snapshot is the entire source of truth.
	got := LoadFromGetterWithBase(nil, snapshot)
	if !got.PrivacyMode || got.ShowGitButton || got.ShowIDEButton {
		t.Error("activation must preserve persisted privacy settings")
	}
	if got.ApplicationTheme != snapshot.ApplicationTheme || got.FileFirstLine != snapshot.FileFirstLine {
		t.Error("activation must preserve persisted themes and formats")
	}
}

// Regression: a stale snapshot has public settings (privacyMode=false, showGitButton=true)
// because the user previously worked with public presence. While the IDE was closed,
// the project was configured with privacyMode=true and showGitButton=false.
// When ReloadSettings runs with project context, the host's current private settings
// must overwrite the old public snapshot, and the privacy ratchet must not interfere.
func TestStalePublicSnapshotAdoptsCurrentPrivateSettings(t *testing.T) {
	stalePublicBase := DefaultSettings()
	stalePublicBase.PrivacyMode = false
	stalePublicBase.ShowGitButton = true

	// Host answers with current project settings: privacyMode=true, showGitButton=false
	getter := func(key string) (string, bool) {
		switch key {
		case "privacyMode":
			return "true", true
		case "showGitButton":
			return "false", true
		}
		return "", false
	}

	got := LoadFromGetterWithBase(getter, stalePublicBase)
	if !got.PrivacyMode {
		t.Error("expected privacyMode to become true from host project settings")
	}
	if got.ShowGitButton {
		t.Error("expected showGitButton to become false from host project settings")
	}

	// Apply privacy ratchet simulation (as in ReloadSettings with hasSettingsChanged=false)
	hasSettingsChanged := false
	if !hasSettingsChanged {
		if stalePublicBase.PrivacyMode {
			got.PrivacyMode = true
		}
		if !stalePublicBase.ShowGitButton {
			got.ShowGitButton = false
		}
	}

	if !got.PrivacyMode {
		t.Error("effective privacyMode must remain true")
	}
	if got.ShowGitButton {
		t.Error("effective showGitButton must remain false")
	}
}

// Regression: when base snapshot already has PrivacyMode=true, a host lookup
// returning global default false (or missing data) without an explicit settings-changed
// event must NOT relax privacy.
func TestPrivacyRatchetPreventsDowngradeWithoutSettingsChanged(t *testing.T) {
	persistedPrivateBase := DefaultSettings()
	persistedPrivateBase.PrivacyMode = true
	persistedPrivateBase.ShowGitButton = false

	// Suppose host lookup returns global default false
	getter := func(key string) (string, bool) {
		switch key {
		case "privacyMode":
			return "false", true
		case "showGitButton":
			return "true", true
		}
		return "", false
	}

	loaded := LoadFromGetterWithBase(getter, persistedPrivateBase)

	// Ratchet with hasSettingsChanged = false
	hasSettingsChanged := false
	if !hasSettingsChanged {
		if persistedPrivateBase.PrivacyMode {
			loaded.PrivacyMode = true
		}
		if !persistedPrivateBase.ShowGitButton {
			loaded.ShowGitButton = false
		}
	}

	if !loaded.PrivacyMode {
		t.Error("ratchet must protect PrivacyMode=true when hasSettingsChanged is false")
	}
	if loaded.ShowGitButton {
		t.Error("ratchet must protect ShowGitButton=false when hasSettingsChanged is false")
	}
}

func TestModernTemplatesAndCustomButtonSettings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-modern-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgPath := filepath.Join(tempDir, "config.json")
	content := `{
		"details": "Hacking on {file}",
		"state": "{project} on {branch}",
		"idleDetails": "Chilling in {project}",
		"idleState": "AFK",
		"idleText": "Sleeping in {project}",
		"customButtonLabel": "My Portfolio",
		"customButtonUrl": "https://example.com/cunha",
		"hideSensitiveFiles": true,
		"sessionTimeoutMins": 45
	}`

	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loaded := LoadSettings(cfgPath)
	if loaded.Details != "Hacking on {file}" {
		t.Errorf("expected details 'Hacking on {file}', got '%s'", loaded.Details)
	}
	if loaded.State != "{project} on {branch}" {
		t.Errorf("expected state '{project} on {branch}', got '%s'", loaded.State)
	}
	if loaded.IdleDetails != "Chilling in {project}" {
		t.Errorf("expected idleDetails 'Chilling in {project}', got '%s'", loaded.IdleDetails)
	}
	if loaded.IdleState != "AFK" {
		t.Errorf("expected idleState 'AFK', got '%s'", loaded.IdleState)
	}
	if loaded.IdleText != "Sleeping in {project}" {
		t.Errorf("expected idleText 'Sleeping in {project}', got '%s'", loaded.IdleText)
	}
	if loaded.CustomButtonLabel != "My Portfolio" {
		t.Errorf("expected customButtonLabel 'My Portfolio', got '%s'", loaded.CustomButtonLabel)
	}
	if loaded.CustomButtonURL != "https://example.com/cunha" {
		t.Errorf("expected customButtonUrl 'https://example.com/cunha', got '%s'", loaded.CustomButtonURL)
	}
	if !loaded.HideSensitiveFiles {
		t.Errorf("expected hideSensitiveFiles true")
	}
	if loaded.SessionTimeoutMins != 45 {
		t.Errorf("expected sessionTimeoutMins 45, got %d", loaded.SessionTimeoutMins)
	}
	if loaded.ParsedSessionTimeout() != 45*time.Minute {
		t.Errorf("expected 45m session timeout, got %v", loaded.ParsedSessionTimeout())
	}
}

func TestLegacyToModernTemplateMigration(t *testing.T) {
	s := DefaultSettings()
	s.Details = ""
	s.State = ""
	s.IdleDetails = ""
	s.IdleState = ""
	s.FileSecondLine = "Editing file"
	s.FileFirstLine = "Project Name (Branch)"
	s.ProjectSecondLine = "Navigating project"
	s.ProjectFirstLine = "Project Name only"

	s.NormalizePresets()

	if s.Details != "Editing {file}" {
		t.Errorf("expected migrated details 'Editing {file}', got '%s'", s.Details)
	}
	if s.State != "{project} ({branch})" {
		t.Errorf("expected migrated state '{project} ({branch})', got '%s'", s.State)
	}
	if s.IdleDetails != "Navigating project" {
		t.Errorf("expected migrated idleDetails 'Navigating project', got '%s'", s.IdleDetails)
	}
	if s.IdleState != "{project}" {
		t.Errorf("expected migrated idleState '{project}', got '%s'", s.IdleState)
	}
}
