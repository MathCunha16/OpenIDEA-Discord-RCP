package app

import (
	"testing"

	"OpenIDEA-Discord-Rcp/pkg/config"
	"OpenIDEA-Discord-Rcp/pkg/template"

	"wit_component/openidea_plugin_editor"
)

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
