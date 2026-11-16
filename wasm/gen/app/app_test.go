package app

import (
	"testing"

	"OpenIDEA-Discord-Rcp/pkg/config"
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
