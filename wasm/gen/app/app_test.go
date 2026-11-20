package app

import (
	"strconv"
	"strings"
	"testing"
	"time"

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
	projectRoot := "/home/user/demo"

	// Case 1: Stale public snapshot in memory, but needsHostReload is true
	app.settings = config.DefaultSettings()
	app.settings.PrivacyMode = false
	app.settings.ShowGitButton = true
	app.needsHostReload = true
	app.verifiedProjectRoot = ""

	eff := app.EffectivePresenceSettings(projectRoot)
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
	app.verifiedProjectRoot = ""

	eff = app.EffectivePresenceSettings(projectRoot)
	if !eff.PrivacyMode || eff.ShowGitButton {
		t.Error("persisted private settings must remain private while needsHostReload is true")
	}

	// Case 3: Missing snapshot (first install defaults in safe mode), needsHostReload is true
	app.settings = config.DefaultSettings()
	app.settings.PrivacyMode = true
	app.settings.ShowGitButton = false
	app.needsHostReload = true
	app.verifiedProjectRoot = ""

	eff = app.EffectivePresenceSettings(projectRoot)
	if !eff.PrivacyMode || eff.ShowGitButton {
		t.Error("missing snapshot must enforce safe mode while needsHostReload is true")
	}

	// Case 4: Verified public project, needsHostReload is false and verifiedProjectRoot matches
	app.settings.PrivacyMode = false
	app.settings.ShowGitButton = true
	app.needsHostReload = false
	app.verifiedProjectRoot = projectRoot

	eff = app.EffectivePresenceSettings(projectRoot)
	if eff.PrivacyMode {
		t.Error("verified public project should keep PrivacyMode=false when needsHostReload is false and project matches")
	}
	if !eff.ShowGitButton {
		t.Error("verified public project should keep ShowGitButton=true when needsHostReload is false and project matches")
	}

	// Case 5: Different unverified project rendered when project A is verified
	differentProject := "/home/user/other-private"
	effOther := app.EffectivePresenceSettings(differentProject)
	if !effOther.PrivacyMode {
		t.Error("verification of project A must NEVER authorize PrivacyMode=false for unverified project B")
	}
	if effOther.ShowGitButton {
		t.Error("verification of project A must NEVER authorize ShowGitButton=true for unverified project B")
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
	projectRoot := "/home/user/demo"
	app.settings = config.DefaultSettings()
	app.settings.PrivacyMode = false
	app.settings.ShowGitButton = true
	app.needsHostReload = true
	app.verifiedProjectRoot = ""

	eff := app.EffectivePresenceSettings(projectRoot)

	// Simulate activation presence update with workspace-project fallback (no active editor)
	ev := &ide.EditorEvent{
		File:        "",
		Language:    "",
		Project:     projectRoot,
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

	// Control: Once verified by the host (needsHostReload=false, verifiedProjectRoot=projectRoot), public presence is permitted
	app.needsHostReload = false
	app.verifiedProjectRoot = projectRoot
	verifiedEff := app.EffectivePresenceSettings(projectRoot)
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

// Regression: P1 Cross-Project Privacy Leak.
// Project A has privacyMode=false / showGitButton=true; project B has privacyMode=true / showGitButton=false.
// When switching between open projects in the same plugin instance, verification of A must NEVER authorize
// public presence for B. Focusing B's file must render strictly in privacy mode.
func TestCrossProjectPrivacyIsolation(t *testing.T) {
	app := newApp()

	projA := "/home/user/public-project"
	projB := "/home/user/private-project"

	publicCfg := config.DefaultSettings()
	publicCfg.PrivacyMode = false
	publicCfg.ShowGitButton = true

	privateCfg := config.DefaultSettings()
	privateCfg.PrivacyMode = true
	privateCfg.ShowGitButton = false

	// 1. Project A is active and verified by host
	app.settings = publicCfg
	app.verifiedProjectRoot = projA
	app.needsHostReload = false

	effA := app.EffectivePresenceSettings(projA)
	if effA.PrivacyMode || !effA.ShowGitButton {
		t.Error("project A should have public presence when verified")
	}

	// 2. Before Project B is verified, querying effective settings for B MUST yield Safe Mode
	unverifiedB := app.EffectivePresenceSettings(projB)
	if !unverifiedB.PrivacyMode || unverifiedB.ShowGitButton {
		t.Error("project B must be in Safe Mode while app is verified only for project A")
	}

	// 3. Render Project B while app still has Project A verified in memory:
	// must NOT leak file, project, branch, or repo button
	evB := &ide.EditorEvent{
		File:        "/home/user/private-project/app.go",
		Language:    "go",
		Project:     projB,
		Branch:      "main",
		PrivacyMode: unverifiedB.PrivacyMode,
	}

	resB := app.engine.Render(evB, false)
	if resB.Details != ide.PrivacyModeText {
		t.Errorf("expected details %q in privacy mode, got %q (leaked active file)", ide.PrivacyModeText, resB.Details)
	}
	if resB.State != "" {
		t.Errorf("expected empty state in privacy mode, got %q (leaked project/branch)", resB.State)
	}

	repoWebURLB := "https://github.com/org/private"
	showRepoButtonB := unverifiedB.ShowGitButton && !unverifiedB.PrivacyMode && repoWebURLB != ""
	if showRepoButtonB {
		t.Error("repository button must NOT be shown for project B")
	}

	// 4. Now simulate project transition in Handle/ReloadSettings: Project B is loaded & verified
	app.settings = privateCfg
	app.verifiedProjectRoot = projB

	effB := app.EffectivePresenceSettings(projB)
	if !effB.PrivacyMode || effB.ShowGitButton {
		t.Error("project B must remain private once verified")
	}

	// 5. Switch back to Project A: Project A becomes unverified until reloaded for A
	unverifiedA := app.EffectivePresenceSettings(projA)
	if !unverifiedA.PrivacyMode || unverifiedA.ShowGitButton {
		t.Error("project A must be treated as unverified Safe Mode while verifiedProjectRoot is project B")
	}

	// Reload Project A
	app.settings = publicCfg
	app.verifiedProjectRoot = projA
	reloadedA := app.EffectivePresenceSettings(projA)
	if reloadedA.PrivacyMode || !reloadedA.ShowGitButton {
		t.Error("project A must return to public once reloaded and verified for A")
	}
}

func TestBuildActivityCustomButtons(t *testing.T) {
	app := newApp()
	projectRoot := "/home/user/myproject"
	app.settings = config.DefaultSettings()
	app.settings.ShowGitButton = true
	app.settings.ShowIDEButton = true
	app.settings.PrivacyMode = false
	app.verifiedProjectRoot = projectRoot
	app.needsHostReload = false

	ev := &ide.EditorEvent{
		File:        "/home/user/myproject/main.go",
		Language:    "go",
		Project:     projectRoot,
		Branch:      "main",
		PrivacyMode: false,
	}

	// Case 1: Repo button + Custom button -> exactly 2 buttons, IDE button omitted
	app.settings.CustomButtonLabel = "Visit Portfolio"
	app.settings.CustomButtonURL = "https://portfolio.example.com"
	act1 := app.buildActivity(ev, "https://github.com/myorg/myproject", false)
	if len(act1.Buttons) != 2 {
		t.Fatalf("expected exactly 2 buttons, got %d", len(act1.Buttons))
	}
	if act1.Buttons[0].Label != "View on GitHub" || act1.Buttons[0].URL != "https://github.com/myorg/myproject" {
		t.Errorf("button 0 mismatch: %+v", act1.Buttons[0])
	}
	if act1.Buttons[1].Label != "Visit Portfolio" || act1.Buttons[1].URL != "https://portfolio.example.com" {
		t.Errorf("button 1 mismatch: %+v", act1.Buttons[1])
	}

	// Case 2: Custom button label truncated to 32 runes
	app.settings.CustomButtonLabel = "This label is definitely longer than thirty-two characters"
	act2 := app.buildActivity(ev, "https://github.com/myorg/myproject", false)
	if len(act2.Buttons[1].Label) != 32 {
		t.Errorf("expected label truncated to 32 runes, got %d: %q", len(act2.Buttons[1].Label), act2.Buttons[1].Label)
	}

	// Case 3: Invalid Custom Button URL (not http/https) should be ignored, falling back to OpenIDEA button
	app.settings.CustomButtonLabel = "Malicious"
	app.settings.CustomButtonURL = "javascript:alert(1)"
	act3 := app.buildActivity(ev, "https://github.com/myorg/myproject", false)
	if len(act3.Buttons) != 2 {
		t.Fatalf("expected 2 buttons (Repo + IDE), got %d", len(act3.Buttons))
	}
	if act3.Buttons[1].Label != "Get OpenIDEA" {
		t.Errorf("expected fallback to 'Get OpenIDEA', got %q", act3.Buttons[1].Label)
	}

	// Case 3b: Plain HTTP URL must be rejected per schema/docs (enforce HTTPS)
	app.settings.CustomButtonLabel = "Insecure"
	app.settings.CustomButtonURL = "http://insecure.example.com"
	actInsecure := app.buildActivity(ev, "https://github.com/myorg/myproject", false)
	if len(actInsecure.Buttons) != 2 || actInsecure.Buttons[1].Label != "Get OpenIDEA" {
		t.Errorf("plain HTTP custom button must be rejected, got buttons: %+v", actInsecure.Buttons)
	}

	// Case 4: No repo button (e.g. no remote URL) -> Custom Button + IDE Button
	app.settings.CustomButtonLabel = "Discord Server"
	app.settings.CustomButtonURL = "https://discord.gg/example"
	act4 := app.buildActivity(ev, "", false)
	if len(act4.Buttons) != 2 {
		t.Fatalf("expected 2 buttons (Custom + IDE), got %d", len(act4.Buttons))
	}
	if act4.Buttons[0].Label != "Discord Server" {
		t.Errorf("expected button 0 'Discord Server', got %q", act4.Buttons[0].Label)
	}
	if act4.Buttons[1].Label != "Get OpenIDEA" {
		t.Errorf("expected button 1 'Get OpenIDEA', got %q", act4.Buttons[1].Label)
	}
}

func TestBuildActivityIdleOmission(t *testing.T) {
	app := newApp()
	projectRoot := "/home/user/myproject"
	app.settings = config.DefaultSettings()
	app.settings.WhileIdle = "show_idling"
	app.settings.ShowGitButton = true
	app.settings.ShowIDEButton = true
	app.verifiedProjectRoot = projectRoot
	app.needsHostReload = false

	ev := &ide.EditorEvent{
		File:        "/home/user/myproject/main.go",
		Language:    "go",
		Project:     projectRoot,
		Branch:      "main",
		PrivacyMode: false,
	}

	// When idle with show_idling, timestamps must be nil and repo button suppressed
	act := app.buildActivity(ev, "https://github.com/myorg/myproject", true)
	if act.Timestamps != nil {
		t.Errorf("expected nil timestamps while idle, got %+v", act.Timestamps)
	}
	for _, b := range act.Buttons {
		if strings.Contains(b.Label, "GitHub") || strings.Contains(b.Label, "Repository") {
			t.Errorf("repo button must be omitted while idle show_idling, got %+v", b)
		}
	}
}

func TestRecordActivityHiatusWithoutIdleTransition(t *testing.T) {
	app := newApp()
	app.settings = config.DefaultSettings()
	app.settings.SessionTimeoutMins = 30 // 30 minutes
	app.settings.DetectIdle = false      // Idle detection disabled

	// Fake an activity 35 minutes ago
	t35m := time.Now().Add(-35 * time.Minute).Unix()
	_ = app.storage.SetString(StorageKeyLastActivity, strconv.FormatInt(t35m, 10))
	_ = app.storage.SetString(StorageKeySessionStart, strconv.FormatInt(t35m, 10))

	// New activity comes in without an idle transition
	app.recordActivity("/home/user/myproject/main.go")

	// Session start should have been reset to approximately now
	sessionStartStr := app.storage.GetString(StorageKeySessionStart)
	start, _ := strconv.ParseInt(sessionStartStr, 10, 64)
	if time.Since(time.Unix(start, 0)) > 5*time.Second {
		t.Errorf("expected session start to reset after hiatus, got start=%v", time.Unix(start, 0))
	}
}

func TestGetSessionStartWithGapDoesNotAdvanceAFKClock(t *testing.T) {
	app := newApp()
	t40m := time.Now().Add(-40 * time.Minute).Unix()
	_ = app.storage.SetString(StorageKeyLastActivity, strconv.FormatInt(t40m, 10))
	_ = app.storage.SetString(StorageKeySessionStart, strconv.FormatInt(t40m, 10))

	// Calling GetSessionStartWithGap during Activate should NOT update lastActivity
	app.storage.GetSessionStartWithGap(30 * 60)

	lastActStr := app.storage.GetString(StorageKeyLastActivity)
	lastAct, _ := strconv.ParseInt(lastActStr, 10, 64)
	if lastAct != t40m {
		t.Errorf("GetSessionStartWithGap advanced lastActivity (AFK clock): got %d, want %d", lastAct, t40m)
	}
}

func TestIdleSensitiveFileTemplateRedaction(t *testing.T) {
	app := newApp()
	projectRoot := "/work/demo"
	app.settings = config.DefaultSettings()
	app.settings.HideSensitiveFiles = true
	app.settings.PrivacyMode = false
	app.settings.IdleText = "Away from {file}"
	app.settings.IdleState = "{file_relative}"
	app.settings.WhileIdle = "show_idling"
	app.verifiedProjectRoot = projectRoot
	app.needsHostReload = false
	applyTemplateSettings(app.engine, app.settings)

	activeFile := "/work/demo/.env.production"
	ev := &ide.EditorEvent{
		File:        activeFile,
		Language:    "env",
		Project:     projectRoot,
		Branch:      "main",
		PrivacyMode: app.settings.PrivacyMode,
		Sensitive:   app.settings.HideSensitiveFiles && ide.IsSensitiveFile(activeFile),
	}

	act := app.buildActivity(ev, "", true)
	if strings.Contains(act.Details, ".env.production") {
		t.Errorf("idle details leaked sensitive file: %q", act.Details)
	}
	if strings.Contains(act.State, ".env.production") {
		t.Errorf("idle state leaked sensitive file relative: %q", act.State)
	}
	if act.Details != "Away from" {
		t.Errorf("expected details 'Away from', got %q", act.Details)
	}
	if act.State != "" {
		t.Errorf("expected empty state when file_relative is redacted, got %q", act.State)
	}
}
