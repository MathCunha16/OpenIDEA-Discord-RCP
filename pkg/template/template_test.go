package template

import (
	"strings"
	"testing"
	"unicode/utf8"

	"OpenIDEA-Discord-Rcp/pkg/ide"
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

	// Test with PrefixReadingEditing disabled
	eng.PrefixReadingEditing = false
	resNoPrefix := eng.Render(ev, false)
	if resNoPrefix.Details != "main.go" {
		t.Errorf("expected details %q without prefix, got %q", "main.go", resNoPrefix.Details)
	}
}

func TestEngineRenderSensitiveAndPrivacy(t *testing.T) {
	eng := NewEngine("material", "material")
	eng.State = "{project} - {file}"

	// Sensitive file (gated by sensitive: true)
	evSensitive := &ide.EditorEvent{
		File:      ".env.production",
		Project:   "MyProject",
		Sensitive: true,
	}
	resSensitive := eng.Render(evSensitive, false)
	if resSensitive.Details != ide.ProtectedConfigText {
		t.Errorf("expected %q, got %q", ide.ProtectedConfigText, resSensitive.Details)
	}
	// State should NOT leak .env.production
	if resSensitive.State != "MyProject" {
		t.Errorf("expected sensitive state to omit file and clean up to 'MyProject', got %q", resSensitive.State)
	}

	// Sensitive file with Sensitive: false (user disabled hideSensitiveFiles)
	evSensitiveDisabled := &ide.EditorEvent{
		File:      ".env.production",
		Project:   "MyProject",
		Sensitive: false,
	}
	resDisabled := eng.Render(evSensitiveDisabled, false)
	if resDisabled.Details == ide.ProtectedConfigText {
		t.Errorf("expected normal rendering when Sensitive is false, got %q", resDisabled.Details)
	}
	if !strings.Contains(resDisabled.State, ".env.production") {
		t.Errorf("expected state to contain file when Sensitive is false, got %q", resDisabled.State)
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
	if resPrivacy.State != "" {
		t.Errorf("expected empty state in privacy mode, got %q", resPrivacy.State)
	}

	// Privacy mode (project only)
	evPrivacyProject := &ide.EditorEvent{
		Project:     "ConfidentialProject",
		PrivacyMode: true,
	}
	resPrivacyProject := eng.Render(evPrivacyProject, false)
	if resPrivacyProject.Details != "In workspace" {
		t.Errorf("expected 'In workspace', got %q", resPrivacyProject.Details)
	}
	if resPrivacyProject.State != "" {
		t.Errorf("expected empty state in privacy mode, got %q", resPrivacyProject.State)
	}
}

func TestEngineRenderIdleAndProjectOnly(t *testing.T) {
	eng := NewEngine("material", "material")
	eng.IdleText = "Idle in {project}"
	eng.IdleState = ""

	// Idle state
	evIdle := &ide.EditorEvent{
		Project: "MyProject",
	}
	resIdle := eng.Render(evIdle, true)
	if resIdle.Details != "Idle in MyProject" {
		t.Errorf("expected details 'Idle in MyProject', got %q", resIdle.Details)
	}
	if resIdle.State != "" {
		t.Errorf("expected empty idle state, got %q", resIdle.State)
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

func TestFormatTemplatePlaceholders(t *testing.T) {
	eng := NewEngine("material", "material")

	ctxWithBranch := Context{
		File:     "main.rs",
		Project:  "my-app",
		Branch:   "feature/login",
		Language: "Rust",
	}

	ctxNoBranch := Context{
		File:     "main.rs",
		Project:  "my-app",
		Branch:   "",
		Language: "Rust",
	}

	// 1. Full interpolation
	res := eng.formatTemplate("Editing {file} in {project} on {branch}", ctxWithBranch)
	if res != "Editing main.rs in my-app on feature/login" {
		t.Errorf("expected full interpolation, got %q", res)
	}

	// 2. Trailing separator cleanup when branch is empty
	resDot := eng.formatTemplate("{project} · {branch}", ctxNoBranch)
	if resDot != "my-app" {
		t.Errorf("expected trailing dot separator stripped, got %q", resDot)
	}

	resDash := eng.formatTemplate("{project} - {branch}", ctxNoBranch)
	if resDash != "my-app" {
		t.Errorf("expected trailing dash separator stripped, got %q", resDash)
	}

	resParens := eng.formatTemplate("{project} ({branch})", ctxNoBranch)
	if resParens != "my-app" {
		t.Errorf("expected empty parens stripped, got %q", resParens)
	}

	// 3. Short strings are padded for Discord minimum length (2 characters)
	resShort := eng.formatTemplate("A", ctxNoBranch)
	if resShort != "A " {
		t.Errorf("expected single char to be padded to 2 chars, got %q (len=%d)", resShort, len(resShort))
	}

	// 4. Overlong strings are truncated to 128 characters
	longTpl := strings.Repeat("x", 200)
	resLong := eng.formatTemplate(longTpl, ctxNoBranch)
	if len(resLong) != 128 {
		t.Errorf("expected 128 chars, got %d", len(resLong))
	}
}
