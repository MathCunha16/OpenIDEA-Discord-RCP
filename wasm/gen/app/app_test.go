package app

import (
	"testing"

	"OpenIDEA-Discord-Rcp/pkg/config"
	"OpenIDEA-Discord-Rcp/pkg/ide"
	"OpenIDEA-Discord-Rcp/pkg/template"

	"wit_component/openidea_plugin_editor"
)

func init() {
	hostAvailable = false
}

func TestAppTemplateSettings(t *testing.T) {
	eng := template.NewEngine("material", "classic")
	cfg := config.DefaultSettings()
	cfg.FileFirstLine = "{file}"
	cfg.FileSecondLine = "{project}"
	cfg.ProjectFirstLine = "{project}"
	cfg.ProjectSecondLine = "Exploring"
	cfg.PrefixReadingEditing = false

	applyTemplateSettings(eng, cfg)

	if eng.FileState != "{file}" {
		t.Errorf("expected FileState '{file}', got %s", eng.FileState)
	}
	if eng.FileDetails != "{project}" {
		t.Errorf("expected FileDetails '{project}', got %s", eng.FileDetails)
	}
	if eng.ProjectState != "{project}" {
		t.Errorf("expected ProjectState '{project}', got %s", eng.ProjectState)
	}
	if eng.ProjectDetails != "Exploring" {
		t.Errorf("expected ProjectDetails 'Exploring', got %s", eng.ProjectDetails)
	}
	if eng.PrefixReadingEditing {
		t.Errorf("expected PrefixReadingEditing false")
	}
}

func TestAppInstance(t *testing.T) {
	app := GetInstance()
	if app == nil {
		t.Fatalf("expected non-nil app instance")
	}
	if app.engine == nil {
		t.Fatalf("expected non-nil engine")
	}
	if app.tracker == nil {
		t.Fatalf("expected non-nil tracker")
	}
	if app.storage == nil {
		t.Fatalf("expected non-nil storage")
	}
	if app.ipc == nil {
		t.Fatalf("expected non-nil ipc")
	}
}

// Regression: the OpenEditors snapshot can lag one tab behind the event that
// triggered the update. The event path must win.
func TestSelectActiveFile(t *testing.T) {
	editors := []openidea_plugin_editor.OpenEditor{
		{Path: "/p/old.go", Language: "go", Active: true},
		{Path: "/p/new.go", Language: "go", Active: false},
	}

	tests := []struct {
		name      string
		eventFile string
		editors   []openidea_plugin_editor.OpenEditor
		wantFile  string
		wantLang  string
	}{
		{
			name:      "event path overrides stale active tab",
			eventFile: "/p/new.go",
			editors:   editors,
			wantFile:  "/p/new.go",
			wantLang:  "go",
		},
		{
			name:     "no event path falls back to snapshot",
			editors:  editors,
			wantFile: "/p/old.go",
			wantLang: "go",
		},
		{
			name:      "event file missing from snapshot keeps path, drops language",
			eventFile: "/p/brand_new.rs",
			editors:   editors,
			wantFile:  "/p/brand_new.rs",
			wantLang:  "",
		},
		{
			name:      "event path with empty snapshot",
			eventFile: "/p/main.go",
			wantFile:  "/p/main.go",
			wantLang:  "",
		},
		{
			name:      "ignored virtual doc event falls back to active snapshot editor",
			eventFile: "{temp_virtual_doc}",
			editors:   editors,
			wantFile:  "/p/old.go",
			wantLang:  "go",
		},
		{
			name:      "ignored openidea settings event falls back to snapshot",
			eventFile: "/home/user/.config/openidea/settings.json",
			editors:   editors,
			wantFile:  "/p/old.go",
			wantLang:  "go",
		},
		{
			name:      "snapshot active editor ignored falls back to empty",
			eventFile: "",
			editors: []openidea_plugin_editor.OpenEditor{
				{Path: "{diff}", Active: true},
			},
			wantFile: "",
			wantLang: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFile, gotLang := selectActiveFile(tt.eventFile, tt.editors)
			if gotFile != tt.wantFile || gotLang != tt.wantLang {
				t.Errorf("selectActiveFile(%q) = (%q, %q), want (%q, %q)",
					tt.eventFile, gotFile, gotLang, tt.wantFile, tt.wantLang)
			}
		})
	}
}

func TestComputeStatusItem(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		paused      bool
		connected   bool
		wantText    string
		wantTooltip string
		wantCmd     string
	}{
		{
			name:        "disabled",
			enabled:     false,
			paused:      false,
			connected:   true,
			wantText:    "○ Discord (Off)",
			wantTooltip: "Discord Rich Presence: Disabled in settings",
			wantCmd:     CommandToggle,
		},
		{
			name:        "paused",
			enabled:     true,
			paused:      true,
			connected:   true,
			wantText:    "○ Discord (Paused)",
			wantTooltip: "Discord Rich Presence: Paused (Click to resume)",
			wantCmd:     CommandToggle,
		},
		{
			name:        "disconnected",
			enabled:     true,
			paused:      false,
			connected:   false,
			wantText:    "○ Discord (Disconnected)",
			wantTooltip: "Discord Rich Presence: Disconnected from Discord client (Click to reconnect)",
			wantCmd:     CommandReconnect,
		},
		{
			name:        "active and connected",
			enabled:     true,
			paused:      false,
			connected:   true,
			wantText:    "● Discord",
			wantTooltip: "Discord Rich Presence: Active (Click to pause)",
			wantCmd:     CommandToggle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, tooltip, cmd := computeStatusItem(tt.enabled, tt.paused, tt.connected)
			if text != tt.wantText {
				t.Errorf("expected text %q, got %q", tt.wantText, text)
			}
			if tooltip != tt.wantTooltip {
				t.Errorf("expected tooltip %q, got %q", tt.wantTooltip, tooltip)
			}
			if cmd != tt.wantCmd {
				t.Errorf("expected command %q, got %q", tt.wantCmd, cmd)
			}
		})
	}
}

func TestProjectSettingsKey(t *testing.T) {
	globalKey := projectSettingsKey("")
	if globalKey != StorageKeySettings {
		t.Errorf("expected global key %q, got %q", StorageKeySettings, globalKey)
	}

	p1 := "/home/user/projectA"
	p2 := "/home/user/projectB"

	k1 := projectSettingsKey(p1)
	k2 := projectSettingsKey(p2)

	if k1 == globalKey || k2 == globalKey {
		t.Errorf("project key collided with global key")
	}
	if k1 == k2 {
		t.Errorf("different projects produced identical keys: %q vs %q", k1, k2)
	}
	if len(k1) > 128 {
		t.Errorf("key length %d exceeds OpenIDEA 128-byte limit", len(k1))
	}

	// Deterministic
	if projectSettingsKey(p1) != k1 {
		t.Errorf("projectSettingsKey is not deterministic")
	}
}

// Regression: P1 privacy issue reported by OpenIDEA maintainer.
// When an old snapshot exists with privacyMode=false and showGitButton=true,
// activation must NOT relax privacy while needsHostReload is true.
// The presence emitted on activation must remain strictly in Safe Mode until
// a current invocation verifies that specific project's effective settings.
func TestEffectivePresenceSettingsSafeMode(t *testing.T) {
	app := newApp()

	// Case 1: Stale public snapshot in memory, but needsHostReload is true
	app.settings = config.DefaultSettings()
	app.settings.PrivacyMode = false
	app.settings.ShowGitButton = true
	app.needsHostReload = true

	eff := app.EffectivePresenceSettings()
	if !eff.PrivacyMode {
		t.Error("stale public snapshot must NOT allow PrivacyMode=false while needsHostReload is true")
	}
	if eff.ShowGitButton {
		t.Error("stale public snapshot must NOT allow ShowGitButton=true while needsHostReload is true")
	}

	// Case 2: Persisted private snapshot, needsHostReload is true
	app.settings.PrivacyMode = true
	app.settings.ShowGitButton = false
	app.needsHostReload = true

	eff = app.EffectivePresenceSettings()
	if !eff.PrivacyMode || eff.ShowGitButton {
		t.Error("persisted private settings must remain private while needsHostReload is true")
	}

	// Case 3: Missing snapshot (first install defaults in safe mode), needsHostReload is true
	app.settings = config.DefaultSettings()
	app.settings.PrivacyMode = true
	app.settings.ShowGitButton = false
	app.needsHostReload = true

	eff = app.EffectivePresenceSettings()
	if !eff.PrivacyMode || eff.ShowGitButton {
		t.Error("missing snapshot must enforce safe mode while needsHostReload is true")
	}

	// Case 4: Verified public project, needsHostReload is false (first project event confirmed it)
	app.settings.PrivacyMode = false
	app.settings.ShowGitButton = true
	app.needsHostReload = false

	eff = app.EffectivePresenceSettings()
	if eff.PrivacyMode {
		t.Error("verified public project should keep PrivacyMode=false when needsHostReload is false")
	}
	if !eff.ShowGitButton {
		t.Error("verified public project should keep ShowGitButton=true when needsHostReload is false")
	}
}

// Reproduction from maintainer:
// "captured state: 'demo (main)', large_text: 'demo', and a View on GitHub button pointing to
// https://github.com/org/demo, despite the current effective project settings being private.
// This does not require an open editor: the workspace-project fallback supplies the project and VCS information."
// Verify that rendering during activation with a stale public snapshot in memory NEVER leaks
// project name, branch, or repository button.
func TestStalePublicSnapshotRendersPrivateOnActivation(t *testing.T) {
	app := newApp()
	app.settings = config.DefaultSettings()
	app.settings.PrivacyMode = false
	app.settings.ShowGitButton = true
	app.needsHostReload = true

	eff := app.EffectivePresenceSettings()

	// Simulate activation presence update with workspace-project fallback (no active editor)
	ev := &ide.EditorEvent{
		File:        "",
		Language:    "",
		Project:     "/home/user/demo",
		Branch:      "main",
		PrivacyMode: eff.PrivacyMode,
	}

	res := app.engine.Render(ev, false)

	// In Safe Mode, project and branch names must be completely redacted
	if res.State != "" {
		t.Errorf("expected empty state in safe mode, got %q (leaked branch/project)", res.State)
	}
	if res.Details != "In workspace" {
		t.Errorf("expected details 'In workspace', got %q", res.Details)
	}
	if res.LargeImageText != "OpenIDEA" {
		t.Errorf("expected large image text 'OpenIDEA', got %q (leaked project name)", res.LargeImageText)
	}

	// Repo button must be suppressed
	repoWebURL := "https://github.com/org/demo"
	showRepoButton := eff.ShowGitButton && !eff.PrivacyMode && repoWebURL != ""
	if showRepoButton {
		t.Errorf("repo button must be omitted during safe mode activation")
	}

	// Control: Once verified by the host (needsHostReload=false), public presence is permitted
	app.needsHostReload = false
	verifiedEff := app.EffectivePresenceSettings()
	ev.PrivacyMode = verifiedEff.PrivacyMode

	verifiedRes := app.engine.Render(ev, false)
	if verifiedRes.State == "" {
		t.Error("expected non-empty state once project is verified public")
	}
	showRepoButtonVerified := verifiedEff.ShowGitButton && !verifiedEff.PrivacyMode && repoWebURL != ""
	if !showRepoButtonVerified {
		t.Error("expected repo button to be permitted once project is verified public")
	}
}
