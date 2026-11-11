package template

import (
	"strings"
	"testing"
	"unicode/utf8"

	"OpenIDEA-Discord-Rcp/internal/ide"
)

func TestFitToRange(t *testing.T) {
	// 1. Minimum length padding with ZeroWidthSpace
	res := FitToRange("a", 2, 128)
	if utf8.RuneCountInString(res) != 2 {
		t.Fatalf("expected length 2, got %d (value: %q)", utf8.RuneCountInString(res), res)
	}
	if !strings.HasPrefix(res, "a") || !strings.Contains(res, ZeroWidthSpace) {
		t.Fatalf("expected string padded with zero-width space, got %q", res)
	}

	// 2. Empty string padding
	emptyRes := FitToRange("", 2, 128)
	if utf8.RuneCountInString(emptyRes) != 2 {
		t.Fatalf("expected length 2 for empty, got %d", utf8.RuneCountInString(emptyRes))
	}

	// 3. Normal range untouched
	normal := "Editing router.py"
	if FitToRange(normal, 2, 128) != normal {
		t.Fatalf("expected %q, got %q", normal, FitToRange(normal, 2, 128))
	}

	// 4. Smooth truncation above max length
	longText := strings.Repeat("A", 140)
	truncated := FitToRange(longText, 2, 128)
	if utf8.RuneCountInString(truncated) != 128 {
		t.Fatalf("expected length 128, got %d", utf8.RuneCountInString(truncated))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Fatalf("expected truncated string to end with ..., got %q", truncated)
	}
}

func TestCDNResolver(t *testing.T) {
	resolver := NewCDNResolver("material", "catppuccin_mocha")
	if resolver.AppTheme() != "material" {
		t.Fatalf("expected appTheme material, got %s", resolver.AppTheme())
	}
	if resolver.LangTheme() != "catppuccin_mocha" {
		t.Fatalf("expected langTheme catppuccin_mocha, got %s", resolver.LangTheme())
	}

	langURL := resolver.LanguageIconURL("python")
	expectedLangURL := "https://raw.githubusercontent.com/MathCunha16/OpenIDEA-Discord-RCP/main/assets/icons/themes/catppuccin_mocha/languages/python.png"
	if langURL != expectedLangURL {
		t.Fatalf("expected %s, got %s", expectedLangURL, langURL)
	}

	appURL := resolver.ApplicationIconURL("openidea")
	expectedAppURL := "https://raw.githubusercontent.com/MathCunha16/OpenIDEA-Discord-RCP/main/assets/icons/themes/material/applications/openidea.png"
	if appURL != expectedAppURL {
		t.Fatalf("expected %s, got %s", expectedAppURL, appURL)
	}

	// Invalid theme fallback to material
	resolver.SetTheme("non_existent_theme")
	if resolver.Theme() != "material" {
		t.Fatalf("expected fallback to material, got %s", resolver.Theme())
	}
}

func TestEngineRenderFile(t *testing.T) {
	eng := NewEngine("material", "material")

	ev := &ide.EditorEvent{
		File:       "/home/user/project/cmd/daemon/main.go",
		Language:   "go",
		Project:    "OpenIDEA-Discord-Rcp",
		Branch:     "feat/template-engine-and-cdn",
		CursorLine: 42,
		TotalLines: 120,
	}

	res := eng.Render(ev, false)

	expectedDetails := "Editing main.go"
	expectedState := "OpenIDEA-Discord-Rcp (feat/template-engine-and-cdn)"

	if res.Details != expectedDetails {
		t.Errorf("expected details %q, got %q", expectedDetails, res.Details)
	}
	if res.State != expectedState {
		t.Errorf("expected state %q, got %q", expectedState, res.State)
	}

	if !strings.Contains(res.LargeImageURL, "languages/go.png") {
		t.Errorf("expected large image URL with go.png, got %s", res.LargeImageURL)
	}
	if res.LargeImageText != "Go" {
		t.Errorf("expected large image text Go, got %s", res.LargeImageText)
	}
}

func TestEngineRenderSensitiveAndPrivacy(t *testing.T) {
	eng := NewEngine("material", "material")

	// Sensitive file
	evSensitive := &ide.EditorEvent{
		File:    ".env.production",
		Project: "MyProject",
	}
	resSensitive := eng.Render(evSensitive, false)
	if resSensitive.Details != ide.ProtectedConfigText {
		t.Errorf("expected %q, got %q", ide.ProtectedConfigText, resSensitive.Details)
	}

	// Privacy mode
	evPrivacy := &ide.EditorEvent{
		File:        "router.py",
		Project:     "MyProject",
		PrivacyMode: true,
	}
	resPrivacy := eng.Render(evPrivacy, false)
	if resPrivacy.Details != ide.PrivacyModeText {
		t.Errorf("expected %q, got %q", ide.PrivacyModeText, resPrivacy.Details)
	}
}

func TestEngineRenderIdleAndProjectOnly(t *testing.T) {
	eng := NewEngine("material", "material")

	// Idle state
	evIdle := &ide.EditorEvent{
		Project: "MyProject",
	}
	resIdle := eng.Render(evIdle, true)
	if resIdle.Details != "Idling" {
		t.Errorf("expected details 'Idling', got %q", resIdle.Details)
	}
	if resIdle.State != "MyProject" {
		t.Errorf("expected state 'MyProject', got %q", resIdle.State)
	}

	// Project only (no file)
	resProject := eng.Render(evIdle, false)
	if resProject.Details != "Navigating project" {
		t.Errorf("expected details 'Navigating project', got %q", resProject.Details)
	}
	if resProject.State != "MyProject" {
		t.Errorf("expected state 'MyProject', got %q", resProject.State)
	}
}
