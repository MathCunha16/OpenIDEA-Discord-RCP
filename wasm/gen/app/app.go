package app

import (
	"fmt"

	"OpenIDEA-Discord-Rcp/pkg/discord"
	"wit_component/openidea_plugin_events"
	"wit_component/openidea_plugin_invocation"
	"wit_component/openidea_plugin_types"

	witTypes "go.bytecodealliance.org/pkg/wit/types"
)

const (
	CommandToggle      = "discord-rpc.toggle"
	CommandReconnect   = "discord-rpc.reconnect"
	CommandInstallHelp = "discord-rpc.install-help"
)

// App manages the lifecycle, storage, and IPC presence coordinator.
type App struct {
	ipc     *IPCClient
	storage *StorageManager
}

var instance = &App{
	ipc:     NewIPCClient(),
	storage: NewStorageManager(),
}

// GetInstance returns the singleton app instance.
func GetInstance() *App {
	return instance
}

// Activate handles OpenIDEA plugin activation, subscribing to IDE events and loading state.
func (a *App) Activate(_ctx openidea_plugin_types.Context) witTypes.Result[witTypes.Unit, openidea_plugin_types.Error] {
	// Subscribe to all editor and lifecycle events required for presence
	_ = openidea_plugin_events.Subscribe("editor-focused")
	_ = openidea_plugin_events.Subscribe("file-opened")
	_ = openidea_plugin_events.Subscribe("file-saved")
	_ = openidea_plugin_events.Subscribe("vcs-changed")
	_ = openidea_plugin_events.Subscribe("settings-changed")

	// Ensure session start timestamp is initialized
	_ = a.storage.GetSessionStart()
	a.storage.RecordActivity()

	return witTypes.Ok[witTypes.Unit, openidea_plugin_types.Error](witTypes.Unit{})
}

// Deactivate cleans up open sockets and resources when the plugin is unloaded or reclaimed.
func (a *App) Deactivate() {
	if a.ipc.IsConnected() {
		// Clear activity before disconnecting
		_ = a.ClearPresence()
		a.ipc.Close()
	}
}

// Run executes plugin commands triggered via Command Palette or Status Bar.
func (a *App) Run(id string, _ctx *openidea_plugin_invocation.Invocation, _arg witTypes.Option[string]) witTypes.Result[witTypes.Option[string], openidea_plugin_types.Error] {
	switch id {
	case CommandToggle:
		paused := !a.storage.IsPaused()
		a.storage.SetPaused(paused)
		if paused {
			_ = a.ClearPresence()
			return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.Some("Discord Rich Presence paused."))
		}
		err := a.EnsureConnection()
		if err != nil {
			return witTypes.Err[witTypes.Option[string], openidea_plugin_types.Error](ConvertError(err))
		}
		return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.Some("Discord Rich Presence resumed."))

	case CommandReconnect:
		a.ipc.Close()
		err := a.EnsureConnection()
		if err != nil {
			return witTypes.Err[witTypes.Option[string], openidea_plugin_types.Error](ConvertError(err))
		}
		sessionStart := a.storage.GetSessionStart()
		_ = a.SetPresence("Developing in OpenIDEA", "WASM Mode (1-Click)", sessionStart)
		return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.Some("Connected to Discord IPC successfully."))

	case CommandInstallHelp:
		return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.Some("OpenIDEA Discord Rich Presence runs 100% natively in WebAssembly (1-Click). No external daemon or manual setup required."))

	default:
		return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.None[string]())
	}
}

// Handle processes incoming IDE events.
func (a *App) Handle(events []openidea_plugin_events.Event, _ctx *openidea_plugin_invocation.Invocation) {
	if a.storage.IsPaused() {
		return
	}

	a.storage.RecordActivity()
	sessionStart := a.storage.GetSessionStart()
	_ = a.SetPresence("Developing in OpenIDEA", "WASM Mode (1-Click)", sessionStart)
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

	// Send OpHandshake (0) using shared tested builder
	frame, err := discord.BuildHandshakeFrame(discord.DefaultApplicationID)
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

// SetPresence sends a SET_ACTIVITY frame to Discord using the shared tested builder.
func (a *App) SetPresence(details, state string, startTimestamp int64) error {
	if err := a.EnsureConnection(); err != nil {
		return err
	}

	act := &discord.Activity{
		Details: details,
		State:   state,
		Timestamps: &discord.Timestamps{
			Start: startTimestamp,
		},
		Assets: &discord.Assets{
			LargeImage: "openidea",
			LargeText:  "OpenIDEA",
		},
	}

	frame, err := discord.BuildActivityFrame(act, 1)
	if err != nil {
		return fmt.Errorf("build activity frame: %w", err)
	}

	if err := a.ipc.SendRawFrame(frame); err != nil {
		return fmt.Errorf("set_activity write: %w", err)
	}

	// Drain response frame from socket
	op, resp, err := a.ipc.ReadFrame(1000)
	if err != nil {
		return fmt.Errorf("set_activity read: %w", err)
	}
	if op == discord.OpClose {
		a.ipc.Close()
		return fmt.Errorf("discord closed connection: %s", string(resp))
	}
	return nil
}

// ClearPresence removes the active activity from Discord.
func (a *App) ClearPresence() error {
	if !a.ipc.IsConnected() {
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
	return nil
}
