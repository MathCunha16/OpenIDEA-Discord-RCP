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
	lastActivity    *discord.Activity
	presenceCleared bool
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
	if cfg.FileFirstLine != "" {
		eng.FileState = cfg.FileFirstLine
	}
	if cfg.FileSecondLine != "" {
		eng.FileDetails = cfg.FileSecondLine
	}
	if cfg.ProjectFirstLine != "" {
		eng.ProjectState = cfg.ProjectFirstLine
	}
	if cfg.ProjectSecondLine != "" {
		eng.ProjectDetails = cfg.ProjectSecondLine
	}
	eng.PrefixReadingEditing = cfg.PrefixReadingEditing
}

// ReloadSettings re-reads settings from OpenIDEA host.
func (a *App) ReloadSettings() {
	getter := func(key string) (string, bool) {
		opt := openidea_plugin_settings.Get(key)
		if opt.IsSome() {
			return opt.Some(), true
		}
		return "", false
	}

	a.settings = config.LoadFromGetter(getter)
	a.tracker.SetIdleTimeout(a.settings.ParsedIdleTimeout())
	a.tracker.SetTimeMode(a.settings.ParsedTimeMode())
	applyTemplateSettings(a.engine, a.settings)

	// Settings changed the rendered payload; force a fresh SET_ACTIVITY even
	// if the last frame happened to compare equal.
	a.lastActivity = nil
	a.presenceCleared = false
	a.updateStatusUI()
}

// Activate handles OpenIDEA plugin activation, subscribing to IDE events and loading state.
func (a *App) Activate(_ctx openidea_plugin_types.Context) witTypes.Result[witTypes.Unit, openidea_plugin_types.Error] {
	// Subscribe to all editor and lifecycle events required for presence
	_ = openidea_plugin_events.Subscribe("editor-focused")
	_ = openidea_plugin_events.Subscribe("file-opened")
	_ = openidea_plugin_events.Subscribe("file-saved")
	_ = openidea_plugin_events.Subscribe("vcs-changed")
	_ = openidea_plugin_events.Subscribe("settings-changed")

	// Load settings from host
	a.ReloadSettings()

	// Restore session state from persistent storage. Do NOT record a fresh
	// activity here: reactivation after an idle reclaim must not reset the AFK
	// clock, otherwise a user who stepped away would immediately look active.
	start := a.storage.GetSessionStart()
	a.tracker.SetProjectStart(start)
	a.tracker.SetLastActivity(a.storage.GetLastActivity())
	if a.settings.DetectIdle {
		a.tracker.CheckIdle(time.Now())
	}

	// Initial presence update if enabled and not paused
	if a.settings.Enabled && !a.storage.IsPaused() {
		_ = a.UpdatePresence("")
	}
	a.updateStatusUI()

	return witTypes.Ok[witTypes.Unit, openidea_plugin_types.Error](witTypes.Unit{})
}

// Deactivate cleans up open sockets and resources when the plugin is unloaded or reclaimed.
func (a *App) Deactivate() {
	if a.ipc.IsConnected() {
		_ = a.ClearPresence()
		a.ipc.Close()
	}
	a.lastActivity = nil
	a.presenceCleared = false
	a.updateStatusUI()
}

// Run executes plugin commands triggered via Command Palette or Status Bar.
func (a *App) Run(id string, _ctx *openidea_plugin_invocation.Invocation, _arg witTypes.Option[string]) witTypes.Result[witTypes.Option[string], openidea_plugin_types.Error] {
	switch id {
	case CommandToggle:
		paused := !a.storage.IsPaused()
		a.storage.SetPaused(paused)
		if paused {
			_ = a.ClearPresence()
			a.lastActivity = nil
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
		a.lastActivity = nil
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
func (a *App) Handle(events []openidea_plugin_events.Event, _ctx *openidea_plugin_invocation.Invocation) {
	// Always process settings changes even if paused, so in-memory settings
	// and UI status bar items remain in sync. Deduplicate per batch.
	hasSettingsChanged := false
	for _, ev := range events {
		if ev.Tag() == openidea_plugin_events.Event_Other {
			ext := ev.Other()
			if ext.Kind == "settings-changed" {
				hasSettingsChanged = true
				break
			}
		}
	}
	if hasSettingsChanged {
		a.ReloadSettings()
	}

	if a.storage.IsPaused() {
		return
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
				if a.tracker.OnActivity(filePath) && a.settings.ResetOpenTimeOnReturn {
					a.tracker.ResetProject()
				}
				a.storage.RecordActivity()
			}

		case openidea_plugin_events.Event_FileOpened:
			filePath := ev.FileOpened()
			if filePath != "" && !ide.IsIgnoredFile(filePath) {
				eventFile = filePath
				if a.tracker.OnActivity(filePath) && a.settings.ResetOpenTimeOnReturn {
					a.tracker.ResetProject()
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
				if a.tracker.OnActivity(filePath) && a.settings.ResetOpenTimeOnReturn {
					a.tracker.ResetProject()
				}
				a.storage.RecordActivity()
			}

		case openidea_plugin_events.Event_VcsChanged:
			if a.tracker.OnActivity("") && a.settings.ResetOpenTimeOnReturn {
				a.tracker.ResetProject()
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
	if a.ipc.IsConnected() {
		return nil
	}

	preferred := a.storage.GetPreferredSocket()
	if err := a.ipc.Connect(preferred); err != nil {
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

// UpdatePresence gathers the latest editor state, renders the template, and dispatches to Discord.
// eventFile is the path carried by the triggering IDE event; when set it takes
// precedence over the host's OpenEditors snapshot, which can lag one tab behind.
func (a *App) UpdatePresence(eventFile string) error {
	if a.storage.IsPaused() || !a.settings.Enabled {
		if a.ipc.IsConnected() {
			_ = a.ClearPresence()
		}
		a.lastActivity = nil
		a.updateStatusUI()
		return nil
	}

	var editors []openidea_plugin_editor.OpenEditor
	if editorsRes := openidea_plugin_editor.OpenEditors(); editorsRes.IsOk() {
		editors = editorsRes.Ok()
	}
	activeFile, activeLanguage := selectActiveFile(eventFile, editors)

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

	isIdle := a.settings.DetectIdle && a.tracker.IsIdle()
	if isIdle && a.settings.WhileIdle == "hide_presence" {
		if a.ipc.IsConnected() {
			_ = a.ClearPresence()
		}
		a.lastActivity = nil
		return nil
	}

	ev := &ide.EditorEvent{
		File:        activeFile,
		Language:    activeLanguage,
		Project:     projectRoot,
		Branch:      branch,
		PrivacyMode: a.settings.PrivacyMode,
	}

	showIdling := isIdle && a.settings.WhileIdle == "show_idling"
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

	if activeFile != "" && !isIdle {
		if a.settings.FileLargeIconRole == "application" {
			largeImgURL = rendered.SmallImageURL
			largeImgText = rendered.SmallImageText
			smallImgURL = rendered.LargeImageURL
			smallImgText = rendered.LargeImageText
		}
		if a.settings.FileSmallIconRole == "none" {
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
	if a.settings.ShowGitButton && !a.settings.PrivacyMode && repoWebURL != "" {
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

	if a.settings.ShowIDEButton {
		buttons = append(buttons, discord.Button{
			Label: "Get OpenIDEA",
			URL:   "https://github.com/l0ng-ai/openidea",
		})
	}

	activity := &discord.Activity{
		Details:    rendered.Details,
		State:      rendered.State,
		Timestamps: timestamps,
		Assets:     assets,
		Buttons:    buttons,
	}

	// Skip network write if activity payload has not changed
	if a.lastActivity != nil && activity.Equal(a.lastActivity) {
		return nil
	}

	return a.sendActivity(activity)
}

// sendActivity sends an Activity frame to Discord.
func (a *App) sendActivity(activity *discord.Activity) error {
	if err := a.EnsureConnection(); err != nil {
		return err
	}

	frame, err := discord.BuildActivityFrame(activity, 1)
	if err != nil {
		return fmt.Errorf("build activity frame: %w", err)
	}

	if err := a.ipc.SendRawFrame(frame); err != nil {
		return fmt.Errorf("set_activity write: %w", err)
	}

	op, resp, err := a.ipc.ReadFrame(1000)
	if err != nil {
		a.ipc.Close()
		a.lastActivity = nil
		a.updateStatusUI()
		return fmt.Errorf("set_activity read: %w", err)
	}
	if op == discord.OpClose {
		a.ipc.Close()
		a.lastActivity = nil
		a.updateStatusUI()
		return fmt.Errorf("discord closed connection: %s", string(resp))
	}

	if op == discord.OpFrame && len(resp) > 0 {
		var incoming discord.IncomingResponse
		if jErr := json.Unmarshal(resp, &incoming); jErr == nil && incoming.Evt == "ERROR" {
			var errDetail discord.ErrorResponse
			if dErr := json.Unmarshal(incoming.Data, &errDetail); dErr == nil && errDetail.Message != "" {
				return fmt.Errorf("discord rejected activity (code %d): %s", errDetail.Code, errDetail.Message)
			}
			return fmt.Errorf("discord rejected activity: %s", string(resp))
		}
	}

	a.lastActivity = activity
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
		return err
	}
	_, _, _ = a.ipc.ReadFrame(500)
	a.lastActivity = nil
	a.presenceCleared = true
	return nil
}
