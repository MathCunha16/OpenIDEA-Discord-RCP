package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"OpenIDEA-Discord-Rcp/pkg/config"
	"OpenIDEA-Discord-Rcp/pkg/discord"
	"OpenIDEA-Discord-Rcp/pkg/git"
	"OpenIDEA-Discord-Rcp/pkg/ide"
	"OpenIDEA-Discord-Rcp/pkg/template"
	"OpenIDEA-Discord-Rcp/pkg/tracker"

	"wit_component/openidea_plugin_editor"
	"wit_component/openidea_plugin_events"
	"wit_component/openidea_plugin_invocation"
	"wit_component/openidea_plugin_log"
	"wit_component/openidea_plugin_settings"
	"wit_component/openidea_plugin_types"
	"wit_component/openidea_plugin_ui"
	"wit_component/openidea_plugin_vcs"
	"wit_component/openidea_plugin_workspace"

	witTypes "go.bytecodealliance.org/pkg/wit/types"
)

const (
	CommandToggle      = "discord-rpc.toggle"
	CommandReconnect   = "discord-rpc.reconnect"
	CommandInstallHelp = "discord-rpc.install-help"
)

// App manages the lifecycle, storage, settings, presence engine, and IPC coordinator.
type App struct {
	ipc             *IPCClient
	storage         *StorageManager
	settings        config.Settings
	tracker         *tracker.SessionTracker
	engine          *template.Engine
	delivery        *discord.DeliveryCache
	presenceCleared bool
	// verifiedProjectRoot is the project whose effective settings are currently
	// loaded and verified in memory. Verification for project A must never
	// authorize public presence for project B.
	verifiedProjectRoot string
	// needsHostReload is set on activation: the first window-scoped event
	// reloads settings from the host, where project overrides are visible.
	needsHostReload bool
}

func newApp() *App {
	cfg := config.DefaultSettings()
	trk := tracker.NewSessionTracker(cfg.ParsedIdleTimeout(), cfg.ParsedTimeMode())
	eng := template.NewEngine(cfg.ApplicationTheme, cfg.LanguageTheme)
	applyTemplateSettings(eng, cfg)

	return &App{
		ipc:      NewIPCClient(),
		storage:  NewStorageManager(),
		settings: cfg,
		tracker:  trk,
		engine:   eng,
		delivery: discord.NewDeliveryCache(),
	}
}

var instance = newApp()

// GetInstance returns the singleton app instance.
func GetInstance() *App {
	return instance
}

var hostAvailable = true

var setStatusItemFn = func(id string, item openidea_plugin_ui.StatusItem) {
	if !hostAvailable {
		return
	}
	openidea_plugin_ui.SetStatusItem(id, item)
}

func computeStatusItem(enabled, paused, connected bool) (string, string, string) {
	if !enabled {
		return "○ Discord (Off)", "Discord Rich Presence: Disabled in settings", CommandToggle
	}
	if paused {
		return "○ Discord (Paused)", "Discord Rich Presence: Paused (Click to resume)", CommandToggle
	}
	if !connected {
		return "○ Discord (Disconnected)", "Discord Rich Presence: Disconnected from Discord client (Click to reconnect)", CommandReconnect
	}
	return "● Discord", "Discord Rich Presence: Active (Click to pause)", CommandToggle
}

func (a *App) updateStatusUI() {
	text, tooltip, command := computeStatusItem(a.settings.Enabled, a.storage.IsPaused(), a.ipc.IsConnected())

	if setStatusItemFn != nil {
		setStatusItemFn("discord-rpc-status", openidea_plugin_ui.StatusItem{
			Text:    text,
			Tooltip: witTypes.Some(tooltip),
			Command: witTypes.Some(command),
		})
	}
}

func applyTemplateSettings(eng *template.Engine, cfg config.Settings) {
	eng.SetThemes(cfg.ApplicationTheme, cfg.LanguageTheme)

	def := config.DefaultSettings()

	fileDetails := cfg.Details
	if cfg.FileSecondLine != def.FileSecondLine && (cfg.Details == "" || cfg.Details == def.Details) {
		fileDetails = cfg.FileSecondLine
	} else if fileDetails == "" {
		fileDetails = cfg.FileSecondLine
	}
	eng.Details = fileDetails
	eng.FileDetails = fileDetails

	fileState := cfg.State
	if cfg.FileFirstLine != def.FileFirstLine && (cfg.State == "" || cfg.State == def.State) {
		fileState = cfg.FileFirstLine
	} else if fileState == "" {
		fileState = cfg.FileFirstLine
	}
	eng.State = fileState
	eng.FileState = fileState

	idleDetails := cfg.IdleDetails
	if cfg.ProjectSecondLine != def.ProjectSecondLine && (cfg.IdleDetails == "" || cfg.IdleDetails == def.IdleDetails) {
		idleDetails = cfg.ProjectSecondLine
	} else if idleDetails == "" {
		idleDetails = cfg.ProjectSecondLine
	}
	eng.IdleDetails = idleDetails
	eng.ProjectDetails = idleDetails

	idleState := cfg.IdleState
	if cfg.ProjectFirstLine != def.ProjectFirstLine && (cfg.IdleState == "" || cfg.IdleState == def.IdleState) {
		idleState = cfg.ProjectFirstLine
	} else if idleState == "" {
		idleState = cfg.ProjectFirstLine
	}
	eng.IdleState = idleState
	eng.ProjectState = idleState

	if cfg.IdleText != "" {
		eng.IdleText = cfg.IdleText
	}
	eng.PrefixReadingEditing = cfg.PrefixReadingEditing
}

// applySettings installs cfg everywhere and forgets cached presence state.
func (a *App) applySettings(cfg config.Settings) {
	a.settings = cfg
	a.tracker.SetIdleTimeout(cfg.ParsedIdleTimeout())
	a.tracker.SetSessionTimeout(cfg.ParsedSessionTimeout())
	a.tracker.SetTimeMode(cfg.ParsedTimeMode())
	applyTemplateSettings(a.engine, cfg)

	// Settings changed the rendered payload; force a fresh SET_ACTIVITY even
	// if the last frame happened to compare equal.
	a.delivery.Invalidate()
	a.presenceCleared = false
	a.updateStatusUI()
}

// detectProjectRoot queries the open project roots from the workspace.
// This works even in Activate() or Scheduled where no window context exists.
func (a *App) detectProjectRoot() string {
	projsRes := openidea_plugin_workspace.Projects()
	if !projsRes.IsOk() {
		return ""
	}
	projs := projsRes.Ok()
	defer func() {
		for _, p := range projs {
			p.Drop()
		}
	}()
	if len(projs) > 0 {
		if rootRes := projs[0].Root(); rootRes.IsOk() {
			return rootRes.Ok()
		}
	}
	return ""
}

// getProjectRoot extracts the project root path from the invocation context.
func getProjectRoot(ctx *openidea_plugin_invocation.Invocation) string {
	if ctx == nil {
		return ""
	}
	projOpt := ctx.Project()
	if projOpt.IsNone() {
		return ""
	}
	proj := projOpt.Some()
	defer proj.Drop()
	if rootRes := proj.Root(); rootRes.IsOk() {
		return rootRes.Ok()
	}
	return ""
}

// hasProjectContext reports whether the invocation carries an open project.
func hasProjectContext(ctx *openidea_plugin_invocation.Invocation) bool {
	return getProjectRoot(ctx) != ""
}

// RestoreSettings is used on activation. OpenIDEA runs activate() with an
// empty Invocation, so settings.Get there resolves only the GLOBAL layer and
// returns schema defaults for any key set at project scope. Consulting the
// host during activation would silently turn Privacy Mode off and leak file,
// project and branch names after the 5-minute idle reclaim.
// We restore the project-scoped snapshot written during a verified event.
// Safe Mode on activation: Until ReloadSettings confirms effective settings
// from the host within a project-scoped invocation, privacy must ALWAYS be
// enforced in memory (PrivacyMode: true, ShowGitButton: false). An old
// snapshot must never authorize relaxing privacy before current project
// settings are verified. We do NOT write this transient state to storage.
func (a *App) RestoreSettings(projectRoot string) {
	cfg, ok := a.storage.LoadSettings(projectRoot)
	if !ok {
		cfg = config.DefaultSettings()
	}
	cfg.PrivacyMode = true
	cfg.ShowGitButton = false
	a.verifiedProjectRoot = ""
	a.applySettings(cfg)
}

// ReloadSettings overlays host settings on the snapshot. Only call it from a
// command or event with project context, where Invocation carries workspace_root
// and project overrides are visible to the host.
func (a *App) ReloadSettings(hasSettingsChanged bool, ctx *openidea_plugin_invocation.Invocation) {
	getter := func(key string) (string, bool) {
		opt := openidea_plugin_settings.Get(key)
		if opt.IsSome() {
			return opt.Some(), true
		}
		return "", false
	}

	projectRoot := getProjectRoot(ctx)
	if projectRoot == "" {
		projectRoot = a.detectProjectRoot()
	}

	base, hasBase := a.storage.LoadSettings(projectRoot)
	if !hasBase {
		base = config.DefaultSettings()
	}

	cfg := config.LoadFromGetterWithBase(getter, base)

	// Strict Privacy Ratchet:
	// If the user has not explicitly triggered a `settings-changed` event,
	// never allow PrivacyMode to flip from true to false, and never allow
	// ShowGitButton to flip from false to true. Only an intentional user
	// settings change can relax privacy.
	if !hasSettingsChanged {
		if base.PrivacyMode {
			cfg.PrivacyMode = true
		}
		if !base.ShowGitButton {
			cfg.ShowGitButton = false
		}
	}

	_ = a.storage.SaveSettings(projectRoot, cfg)
	a.verifiedProjectRoot = projectRoot
	a.applySettings(cfg)
}

// Activate handles OpenIDEA plugin activation, subscribing to IDE events and loading state.
func (a *App) Activate(_ctx openidea_plugin_types.Context) witTypes.Result[witTypes.Unit, openidea_plugin_types.Error] {
	// Subscribe to all editor and lifecycle events required for presence
	_ = openidea_plugin_events.Subscribe("editor-focused")
	_ = openidea_plugin_events.Subscribe("file-opened")
	_ = openidea_plugin_events.Subscribe("file-saved")
	_ = openidea_plugin_events.Subscribe("vcs-changed")
	_ = openidea_plugin_events.Subscribe("settings-changed")

	// Restore persisted settings only. The host is deliberately NOT consulted
	// here: activate() has no workspace_root, so it would return global
	// defaults for project-scoped keys (see RestoreSettings).
	projectRoot := a.detectProjectRoot()
	a.RestoreSettings(projectRoot)
	a.needsHostReload = true

	// Restore session state from persistent storage using the dynamic session gap.
	// Reactivation after an idle reclaim must not reset the AFK clock, but hiatus
	// exceeding session timeout (e.g. 30m or PC reboot) initializes a fresh session.
	gap := int64(a.settings.ParsedSessionTimeout().Seconds())
	if a.settings.ResetOpenTimeOnReturn {
		gap = int64(a.settings.ParsedIdleTimeout().Seconds())
	}
	start := a.storage.GetSessionStartWithGap(gap)
	a.tracker.SetProjectStart(start)
	a.tracker.SetLastActivity(a.storage.GetLastActivity())
	if a.settings.DetectIdle {
		a.tracker.CheckIdle(time.Now())
	}

	// Reconnect immediately on activation. The host reclaims idle instances
	// after 5 minutes and Deactivate closes the socket; without this the user
	// returns from AFK to a "Disconnected" status until a manual command.
	if a.settings.Enabled && !a.storage.IsPaused() {
		if err := a.EnsureConnection(); err != nil {
			// The next event retries via UpdatePresence; log so the failure is
			// visible instead of being silently swallowed.
			openidea_plugin_log.Log(openidea_plugin_types.LogLevel_Warn, "discord-rpc: activation reconnect failed: "+err.Error())
		} else {
			_ = a.UpdatePresence("")
		}
	}
	a.updateStatusUI()

	return witTypes.Ok[witTypes.Unit, openidea_plugin_types.Error](witTypes.Unit{})
}

// Deactivate cleans up open sockets when the plugin is unloaded or reclaimed.
// It deliberately does NOT clear the Discord activity: the host reclaims idle
// instances after 5 minutes, and wiping the presence there makes the user look
// disconnected until a manual reconnect. Discord itself drops the activity when
// the IPC socket closes, and Activate/Handle republish it on the next event.
func (a *App) Deactivate() {
	a.ipc.Close()
	a.delivery.Invalidate()
	a.presenceCleared = false
}

// Run executes plugin commands triggered via Command Palette or Status Bar.
func (a *App) Run(id string, ctx *openidea_plugin_invocation.Invocation, _arg witTypes.Option[string]) witTypes.Result[witTypes.Option[string], openidea_plugin_types.Error] {
	// A command with a project context can refresh the snapshot before any
	// editor event arrives after activation, or when switching project context.
	projectRoot := getProjectRoot(ctx)
	if projectRoot != "" && (projectRoot != a.verifiedProjectRoot || a.needsHostReload) {
		a.ReloadSettings(false, ctx)
		a.needsHostReload = false
	}
	switch id {
	case CommandToggle:
		paused := !a.storage.IsPaused()
		a.storage.SetPaused(paused)
		if paused {
			_ = a.ClearPresence()
			a.delivery.Invalidate()
			a.updateStatusUI()
			return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.Some("Discord Rich Presence paused."))
		}
		err := a.EnsureConnection()
		if err != nil {
			a.updateStatusUI()
			return witTypes.Err[witTypes.Option[string], openidea_plugin_types.Error](ConvertError(err))
		}
		_ = a.UpdatePresence("")
		a.updateStatusUI()
		return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.Some("Discord Rich Presence resumed."))

	case CommandReconnect:
		a.ipc.Close()
		a.delivery.Invalidate()
		err := a.EnsureConnection()
		a.updateStatusUI()
		if err != nil {
			return witTypes.Err[witTypes.Option[string], openidea_plugin_types.Error](ConvertError(err))
		}
		if a.storage.IsPaused() {
			return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.Some("Connected to Discord IPC (Presence remains paused)."))
		}
		_ = a.UpdatePresence("")
		return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.Some("Connected to Discord IPC successfully."))

	case CommandInstallHelp:
		return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.Some("OpenIDEA Discord Rich Presence runs 100% natively in WebAssembly (1-Click). No external daemon or manual setup required."))

	default:
		return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.None[string]())
	}
}

// Handle processes incoming IDE events.
func (a *App) Handle(events []openidea_plugin_events.Event, ctx *openidea_plugin_invocation.Invocation) {
	// Always process settings changes even if paused, so in-memory settings
	// and UI status bar items remain in sync. Deduplicate per batch.
	hasSettingsChanged := false
	hasActivity := false
	for _, ev := range events {
		switch ev.Tag() {
		case openidea_plugin_events.Event_Other:
			ext := ev.Other()
			if ext.Kind == "settings-changed" {
				hasSettingsChanged = true
			}
		case openidea_plugin_events.Event_EditorFocused,
			openidea_plugin_events.Event_FileOpened,
			openidea_plugin_events.Event_FileSaved,
			openidea_plugin_events.Event_VcsChanged:
			hasActivity = true
		}
	}
	// Host settings are only trusted with a project context: without it the
	// host answers with global values, which would overwrite the snapshot's
	// project-scoped choices (e.g. Privacy Mode). Keep waiting for an event
	// that carries the window's project.
	// When switching between open projects in the same instance (projectRoot != a.verifiedProjectRoot),
	// settings must be refreshed for that specific project.
	projectRoot := getProjectRoot(ctx)
	if projectRoot != "" && (projectRoot != a.verifiedProjectRoot || hasSettingsChanged || a.needsHostReload) {
		a.ReloadSettings(hasSettingsChanged, ctx)
		a.needsHostReload = false
	}

	if a.storage.IsPaused() {
		return
	}

	// Scheduled triggers wake inactive instances without client window or open editors.
	// Don't attempt presence updates on a bare Scheduled pulse without editor activity.
	if !hasActivity && !hasSettingsChanged {
		for _, ev := range events {
			if ev.Tag() == openidea_plugin_events.Event_Scheduled {
				return
			}
		}
	}

	now := time.Now()
	// The event payload carries the file the user just moved to. The host's
	// OpenEditors() snapshot can still point at the previous tab, so prefer
	// this path over the snapshot when rendering.
	eventFile := ""
	for _, ev := range events {
		switch ev.Tag() {
		case openidea_plugin_events.Event_EditorFocused:
			opt := ev.EditorFocused()
			var filePath string
			if opt.IsSome() {
				filePath = opt.Some()
			}
			if filePath != "" && !ide.IsIgnoredFile(filePath) {
				eventFile = filePath
				if a.tracker.OnActivity(filePath) {
					gap := a.settings.ParsedSessionTimeout()
					if a.settings.ResetOpenTimeOnReturn {
						gap = a.settings.ParsedIdleTimeout()
					}
					if a.settings.ResetOpenTimeOnReturn || time.Since(time.Unix(a.storage.GetLastActivity(), 0)) >= gap {
						a.tracker.ResetProject()
						_ = a.storage.ResetSessionStart()
					}
				}
				a.storage.RecordActivity()
			}

		case openidea_plugin_events.Event_FileOpened:
			filePath := ev.FileOpened()
			if filePath != "" && !ide.IsIgnoredFile(filePath) {
				eventFile = filePath
				if a.tracker.OnActivity(filePath) {
					gap := a.settings.ParsedSessionTimeout()
					if a.settings.ResetOpenTimeOnReturn {
						gap = a.settings.ParsedIdleTimeout()
					}
					if a.settings.ResetOpenTimeOnReturn || time.Since(time.Unix(a.storage.GetLastActivity(), 0)) >= gap {
						a.tracker.ResetProject()
						_ = a.storage.ResetSessionStart()
					}
				}
				a.storage.RecordActivity()
			}

		case openidea_plugin_events.Event_FileSaved:
			// Saving a file resets the idle timer, but does NOT override the active
			// focused tab snapshot (e.g. Save All saving background tabs).
			// Background saves of ignored files (e.g. internal OpenIDEA files or diffs)
			// must not renew the idle activity clock.
			filePath := ev.FileSaved()
			if filePath != "" && !ide.IsIgnoredFile(filePath) {
				if a.tracker.OnActivity(filePath) {
					gap := a.settings.ParsedSessionTimeout()
					if a.settings.ResetOpenTimeOnReturn {
						gap = a.settings.ParsedIdleTimeout()
					}
					if a.settings.ResetOpenTimeOnReturn || time.Since(time.Unix(a.storage.GetLastActivity(), 0)) >= gap {
						a.tracker.ResetProject()
						_ = a.storage.ResetSessionStart()
					}
				}
				a.storage.RecordActivity()
			}

		case openidea_plugin_events.Event_VcsChanged:
			if a.tracker.OnActivity("") {
				gap := a.settings.ParsedSessionTimeout()
				if a.settings.ResetOpenTimeOnReturn {
					gap = a.settings.ParsedIdleTimeout()
				}
				if a.settings.ResetOpenTimeOnReturn || time.Since(time.Unix(a.storage.GetLastActivity(), 0)) >= gap {
					a.tracker.ResetProject()
					_ = a.storage.ResetSessionStart()
				}
			}
			a.storage.RecordActivity()
		}
	}

	// Idle is derived from the persisted last-activity timestamp, not from a
	// background ticker: the host only runs the plugin while it is delivering
	// events, and does not send Scheduled events to an active instance.
	// Re-sync the tracker from storage so both clocks agree even for events
	// that do not carry a file path (e.g. settings-changed).
	if a.settings.DetectIdle {
		a.tracker.SetLastActivity(a.storage.GetLastActivity())
		a.tracker.CheckIdle(now)
	}

	// Unconditionally update presence: if settings.Enabled is false, UpdatePresence
	// will properly clear any active Discord presence.
	_ = a.UpdatePresence(eventFile)
}

// EnsureConnection connects and performs handshake if disconnected.
func (a *App) EnsureConnection() error {
	a.delivery.SetConnected(a.ipc.IsConnected())
	if a.ipc.IsConnected() {
		return nil
	}

	preferred := a.storage.GetPreferredSocket()
	if err := a.ipc.Connect(preferred); err != nil {
		a.updateStatusUI()
		return err
	}

	appID := a.settings.ApplicationID
	if appID == "" {
		appID = discord.DefaultApplicationID
	}

	frame, err := discord.BuildHandshakeFrame(appID)
	if err != nil {
		return fmt.Errorf("handshake frame: %w", err)
	}
	if err := a.ipc.SendRawFrame(frame); err != nil {
		return fmt.Errorf("handshake write: %w", err)
	}

	op, resp, err := a.ipc.ReadFrame(1000)
	if err != nil {
		return fmt.Errorf("handshake read: %w", err)
	}
	if op != discord.OpFrame {
		a.ipc.Close()
		return fmt.Errorf("handshake: expected op 1, got %d (payload: %s)", op, string(resp))
	}

	a.storage.SavePreferredSocket(a.ipc.LastSocket())
	a.delivery.SetConnected(true)
	return nil
}

// queryVCS retrieves the active branch and normalized remote web URL for a project.
func queryVCS(proj *openidea_plugin_workspace.Project) (string, string) {
	var branch, repoURL string
	if statusRes := openidea_plugin_vcs.Status(proj); statusRes.IsOk() {
		if optStatus := statusRes.Ok(); optStatus.IsSome() {
			st := optStatus.Some()
			if st.Branch.IsSome() {
				branch = st.Branch.Some()
			}
		}
	}

	if remotesRes := openidea_plugin_vcs.Remotes(proj); remotesRes.IsOk() {
		remotes := remotesRes.Ok()
		var selectedURL string
		for _, r := range remotes {
			if r.F0 == "origin" {
				selectedURL = r.F1
				break
			}
		}
		if selectedURL == "" && len(remotes) > 0 {
			selectedURL = remotes[0].F1
		}
		if selectedURL != "" {
			repoURL = git.NormalizeToHTTPS(selectedURL)
		}
	}
	return branch, repoURL
}

// selectActiveFile resolves the file to display. An event path wins over the
// OpenEditors snapshot because that snapshot can still point at the previous
// tab when a focus/open event is delivered.
func selectActiveFile(eventFile string, editors []openidea_plugin_editor.OpenEditor) (string, string) {
	if ide.IsIgnoredFile(eventFile) {
		eventFile = ""
	}

	activeFile, activeLanguage := "", ""
	for _, ed := range editors {
		if ed.Active && !ide.IsIgnoredFile(ed.Path) {
			activeFile, activeLanguage = ed.Path, ed.Language
			break
		}
	}
	if eventFile == "" {
		return activeFile, activeLanguage
	}
	// Adopt the snapshot language only when it belongs to this exact file;
	// otherwise DetectLanguage derives it from the extension.
	for _, ed := range editors {
		if ed.Path == eventFile {
			return eventFile, ed.Language
		}
	}
	return eventFile, ""
}

// EffectivePresenceSettings returns the settings to use for presence rendering.
// It is strictly bound to the specific projectRoot being rendered:
// If needsHostReload is true, or if projectRoot is unverified (empty or does not
// match verifiedProjectRoot), Safe Mode (PrivacyMode: true, ShowGitButton: false)
// is unconditionally enforced. Verification of project A must NEVER authorize
// public presence for project B.
func (a *App) EffectivePresenceSettings(projectRoot string) config.Settings {
	s := a.settings
	if a.needsHostReload || projectRoot == "" || projectRoot != a.verifiedProjectRoot {
		s.PrivacyMode = true
		s.ShowGitButton = false
	}
	return s
}

// UpdatePresence gathers the latest editor state, renders the template, and dispatches to Discord.
// eventFile is the path carried by the triggering IDE event; when set it takes
// precedence over the host's OpenEditors snapshot, which can lag one tab behind.
func (a *App) UpdatePresence(eventFile string) error {
	if a.storage.IsPaused() {
		if a.ipc.IsConnected() {
			_ = a.ClearPresence()
		}
		a.delivery.Invalidate()
		a.updateStatusUI()
		return nil
	}

	var editors []openidea_plugin_editor.OpenEditor
	if editorsRes := openidea_plugin_editor.OpenEditors(); editorsRes.IsOk() {
		editors = editorsRes.Ok()
	}
	activeFile, activeLanguage := selectActiveFile(eventFile, editors)
	openFiles := make([]string, 0, len(editors))
	for _, ed := range editors {
		if ed.Path != "" && !ide.IsIgnoredFile(ed.Path) {
			openFiles = append(openFiles, ed.Path)
		}
	}
	if activeFile != "" {
		found := false
		for _, p := range openFiles {
			if p == activeFile {
				found = true
				break
			}
		}
		if !found {
			openFiles = append(openFiles, activeFile)
		}
	}

	var projectRoot, branch, repoWebURL string
	if activeFile != "" {
		if projOpt := openidea_plugin_workspace.ProjectOf(activeFile); projOpt.IsSome() {
			proj := projOpt.Some()
			if rootRes := proj.Root(); rootRes.IsOk() {
				projectRoot = rootRes.Ok()
			}
			branch, repoWebURL = queryVCS(proj)
			proj.Drop()
		}
	}

	if projectRoot == "" {
		if projsRes := openidea_plugin_workspace.Projects(); projsRes.IsOk() {
			projs := projsRes.Ok()
			for i, p := range projs {
				if i == 0 {
					if rootRes := p.Root(); rootRes.IsOk() {
						projectRoot = rootRes.Ok()
					}
					branch, repoWebURL = queryVCS(p)
				}
				p.Drop()
			}
		}
	}

	effSettings := a.EffectivePresenceSettings(projectRoot)
	if !effSettings.Enabled {
		if a.ipc.IsConnected() {
			_ = a.ClearPresence()
		}
		a.delivery.Invalidate()
		a.updateStatusUI()
		return nil
	}

	isIdle := effSettings.DetectIdle && a.tracker.IsIdle()
	if isIdle && effSettings.WhileIdle == "hide_presence" {
		if a.ipc.IsConnected() {
			_ = a.ClearPresence()
		}
		a.delivery.Invalidate()
		return nil
	}

	ev := &ide.EditorEvent{
		File:        activeFile,
		Language:    activeLanguage,
		Project:     projectRoot,
		Branch:      branch,
		PrivacyMode: effSettings.PrivacyMode,
		Sensitive:   effSettings.HideSensitiveFiles && ide.IsSensitiveFile(activeFile),
		OpenFiles:   openFiles,
	}

	activity := a.buildActivity(ev, repoWebURL, isIdle)

	// A cached payload is only trusted while the IPC connection is valid.
	if a.delivery.ShouldSkip(activity) {
		return nil
	}

	return a.sendActivity(activity)
}

func (a *App) buildActivity(ev *ide.EditorEvent, repoWebURL string, isIdle bool) *discord.Activity {
	effSettings := a.EffectivePresenceSettings(ev.Project)
	showIdling := isIdle && effSettings.WhileIdle == "show_idling"
	rendered := a.engine.Render(ev, showIdling)

	var timestamps *discord.Timestamps
	startTime := a.tracker.TimestampStart()
	if startTime > 0 && !isIdle {
		timestamps = &discord.Timestamps{
			Start: startTime,
		}
	}

	largeImgURL := rendered.LargeImageURL
	largeImgText := rendered.LargeImageText
	smallImgURL := rendered.SmallImageURL
	smallImgText := rendered.SmallImageText

	if ev.File != "" && !isIdle {
		if effSettings.FileLargeIconRole == "application" {
			largeImgURL = rendered.SmallImageURL
			largeImgText = rendered.SmallImageText
			smallImgURL = rendered.LargeImageURL
			smallImgText = rendered.LargeImageText
		}
		if effSettings.FileSmallIconRole == "none" {
			smallImgURL = ""
			smallImgText = ""
		}
	}

	assets := &discord.Assets{
		LargeImage: largeImgURL,
		LargeText:  largeImgText,
		SmallImage: smallImgURL,
		SmallText:  smallImgText,
	}

	var buttons []discord.Button
	// While showing the idle state, the repository link is omitted: an idle
	// presence should not expose the project repository. keep_showing keeps the
	// previous activity untouched, so only show_idling suppresses it.
	showRepoButton := effSettings.ShowGitButton && !effSettings.PrivacyMode && repoWebURL != "" &&
		!(isIdle && effSettings.WhileIdle == "show_idling")
	if showRepoButton {
		label := "View Repository"
		if strings.Contains(repoWebURL, "github.com") {
			label = "View on GitHub"
		} else if strings.Contains(repoWebURL, "gitlab.com") {
			label = "View on GitLab"
		}
		buttons = append(buttons, discord.Button{
			Label: label,
			URL:   repoWebURL,
		})
	}

	customLabel := strings.TrimSpace(effSettings.CustomButtonLabel)
	customURL := strings.TrimSpace(effSettings.CustomButtonURL)
	if customLabel != "" && (strings.HasPrefix(customURL, "https://") || strings.HasPrefix(customURL, "http://")) && len(buttons) < 2 {
		runes := []rune(customLabel)
		if len(runes) > 32 {
			runes = runes[:32]
		}
		buttons = append(buttons, discord.Button{
			Label: string(runes),
			URL:   customURL,
		})
	}

	if effSettings.ShowIDEButton && len(buttons) < 2 {
		buttons = append(buttons, discord.Button{
			Label: "Get OpenIDEA",
			URL:   "https://github.com/l0ng-ai/openidea",
		})
	}

	return &discord.Activity{
		Details:    rendered.Details,
		State:      rendered.State,
		Timestamps: timestamps,
		Assets:     assets,
		Buttons:    buttons,
	}
}

// invalidateDeliveryState forgets the last accepted payload and refreshes the
// status bar after any failed delivery. Without this, a later event whose
// payload matches the stale cache would be deduplicated and never reconnect.
func (a *App) invalidateDeliveryState() {
	a.delivery.Invalidate()
	a.presenceCleared = false
	a.updateStatusUI()
}

// sendActivity sends an Activity frame to Discord.
func (a *App) sendActivity(activity *discord.Activity) error {
	if err := a.EnsureConnection(); err != nil {
		a.updateStatusUI()
		return err
	}

	frame, err := discord.BuildActivityFrame(activity, 1)
	if err != nil {
		return fmt.Errorf("build activity frame: %w", err)
	}

	if err := a.ipc.SendRawFrame(frame); err != nil {
		// SendRawFrame already closed the connection; drop the cached activity
		// so the next event forces a reconnect instead of being skipped.
		a.invalidateDeliveryState()
		return fmt.Errorf("set_activity write: %w", err)
	}

	op, resp, err := a.ipc.ReadFrame(1000)
	if err != nil {
		a.ipc.Close()
		a.invalidateDeliveryState()
		return fmt.Errorf("set_activity read: %w", err)
	}
	if op == discord.OpClose {
		a.ipc.Close()
		a.invalidateDeliveryState()
		return fmt.Errorf("discord closed connection: %s", string(resp))
	}

	if op == discord.OpFrame && len(resp) > 0 {
		var incoming discord.IncomingResponse
		if jErr := json.Unmarshal(resp, &incoming); jErr == nil && incoming.Evt == "ERROR" {
			var errDetail discord.ErrorResponse
			if dErr := json.Unmarshal(incoming.Data, &errDetail); dErr == nil && errDetail.Message != "" {
				a.invalidateDeliveryState()
				return fmt.Errorf("discord rejected activity (code %d): %s", errDetail.Code, errDetail.Message)
			}
			a.invalidateDeliveryState()
			return fmt.Errorf("discord rejected activity: %s", string(resp))
		}
	}

	a.delivery.Delivered(activity)
	a.presenceCleared = false
	a.updateStatusUI()
	return nil
}

// ClearPresence removes the active activity from Discord.
func (a *App) ClearPresence() error {
	if !a.ipc.IsConnected() || a.presenceCleared {
		return nil
	}

	frame, err := discord.BuildActivityFrame(nil, 1)
	if err != nil {
		return err
	}

	if err := a.ipc.SendRawFrame(frame); err != nil {
		a.delivery.Invalidate()
		return err
	}
	_, _, _ = a.ipc.ReadFrame(500)
	a.delivery.Invalidate()
	a.presenceCleared = true
	return nil
}
