<p align="center">
  <img src="assets/logo.png" alt="OpenIDEA Discord Rich Presence" width="200"/>
</p>
<h1 align="center">OpenIDEA Discord Rich Presence</h1>

<p align="center">
  A fast, lightweight, and privacy-conscious Discord Rich Presence plugin for <a href="https://github.com/l0ng-ai/openidea">OpenIDEA</a>. Automatically displays your active project, file details, language icons, Git branch, and elapsed session timer on your Discord profile — 100% natively in WebAssembly with zero external daemons!
</p>

<p align="center">
  <img src="https://img.shields.io/badge/OpenIDEA-%5E1.4-orange" alt="OpenIDEA Engine"/>
  <img src="https://img.shields.io/badge/license-Apache--2.0-blue.svg" alt="License"/>
  <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go" alt="Go"/>
  <img src="https://img.shields.io/badge/WebAssembly-Component%20Model-654FF0?logo=webassembly" alt="WebAssembly Component Model"/>
</p>

---

## Features

- **1-Click Native Presence (Zero Daemon Setup):** Runs 100% inside OpenIDEA as an idiomatic WebAssembly Component Model plugin (`wasip2`). No external companion daemons, background binaries, or `$PATH` management required.
- **Event-Driven & Instant Sync:** Updates presence seamlessly on editor focus, file navigation, VCS branch changes, and IDE settings adjustments.
- **Ultra-Lightweight Footprint:** Sandboxed WebAssembly execution with minimal memory usage and **0.0% idle CPU**.
- **Exhaustive Language & File Recognition:** Parity with 250+ language definitions covering over **475 file extensions** (including composite extensions like `.tar.gz`, `.ng.html`, `.component.ts`, `.spec.js`) and **105 exact file manifests** (`Dockerfile`, `Cargo.toml`, `Makefile`, etc.).
- **6 Rich Icon Themes:**
  - **Material** (default)
  - **Classic**
  - **Atom**
  - **Catppuccin Mocha**
  - **Catppuccin Latte**
  - **Flat**
- **Interactive Status Bar Widget:** Real-time connection feedback with 1-click toggle directly from OpenIDEA's bottom bar (`● Discord` / `○ Discord (Paused)` / `✕ Discord (Disconnected)` / `◌ Discord (Off)`).
- **Customizable Presence Content:** Choose custom line formats for active files and projects, toggle "Editing " prefixes, and pick between project time, file open time, or no timer.
- **Inactivity & AFK Detection:** Derives idle state from a persisted activity timestamp so it survives plugin restarts. OpenIDEA reclaims idle plugin instances after 5 minutes; the plugin reconnects automatically on reactivation without clearing your presence or resetting the AFK clock. See [Idle & AFK behavior](#idle--afk-behavior).
- **Privacy & Security First:**
  - **Privacy Mode:** Completely obfuscates file, project, and branch names from Discord with a single toggle.
  - **Sensitive File Protection:** Automatic sanitization of `.env`, private keys, secrets, and credential files.
  - **No Outbound Internet Traffic:** Only communicates locally with your desktop Discord client over local IPC sockets.
  - **Optional Buttons:** Choose whether to expose your public Git repository or IDE download link on Discord.

---

## Installation & Quickstart

### 1-Click Install

1. Make sure the **Discord desktop app** is running on your machine.
2. In OpenIDEA, open **Settings › Plugins**.
3. Install **Discord Rich Presence** directly from the OpenIDEA Plugin Marketplace, or choose **Install Plugin from Disk...** and select `cunha.discord-rpc-<VERSION>.oiplugin`.
4. Allow the requested capabilities (`ipc:...`, `editor.read`, `project.read`, `vcs.read`).
5. **Done!** Rich Presence activates immediately, and the status bar displays **`● Discord`**.

---

## Supported Platforms & IPC Transports

The plugin connects natively to local Discord IPC sockets across all major desktop environments:

| Platform | Discord IPC Transport |
| --- | --- |
| **Linux (Native)** | Unix domain sockets (`$XDG_RUNTIME_DIR/discord-ipc-N`, `/tmp/discord-ipc-N`) |
| **Linux (Flatpak / Snap / Vesktop)** | Portals and sandboxed paths (`$XDG_RUNTIME_DIR/app/com.discordapp.Discord/`, `snap.discord/`, `dev.vencord.Vesktop/`, etc.) |
| **macOS** | Unix domain sockets (`$TMPDIR/discord-ipc-N`, `/tmp/discord-ipc-N`) |
| **Windows** | Windows Named Pipes (`\\.\pipe\discord-ipc-N`) |

---

## Data Flow & Privacy Disclosure

Transparency is paramount. Here is how data flows through the plugin:

```
[OpenIDEA Editor]
       │
       ▼ (1) IDE Events (Active file path, project root, git branch, settings)
[WASM Component] (wasm32-wasip2 / componentize-go)
       │
       ├─► (2) In-Memory Sanitization & Privacy Filter
       │
       ▼ (3) Sandboxed Local IPC Handshake & SET_ACTIVITY payload
[Discord Desktop Client] (Local IPC Socket / Named Pipe)
```

1. **What the plugin inspects locally:**
   - Active file relative and absolute paths (for language detection and tab switching).
   - Project directory path, plus Git branch and public remote URL reported by OpenIDEA's `vcs.read` (no `.git` files are read directly).
   - User settings (theme, privacy toggle, format preferences).

2. **What the plugin sanitizes:**
   - **Privacy Mode:** When enabled, replaces project and file names with generic descriptions (`Working on a project`).
   - **Sensitive File Filter:** Automatically redacts `.env`, certificates, private keys, passwords, credentials, and virtual IDE buffers (`{diff}`, `{terminal}`).
   - **Git Credentials Protection:** Strips any username/password tokens from repository remote URLs before formatting buttons.
   - **No Outbound Network Access:** Opens no network listeners and makes zero HTTP requests. It connects exclusively to the local Discord client.

3. **What is sent to Discord:**
   - Details text (e.g. `Project Name (main)`).
   - State text (e.g. `Editing main.go`).
   - Asset keys (Discord CDN icon identifiers for OpenIDEA and programming languages).
   - Timestamps (session start time or file open time).
   - Public Git repository URL button (if Git remote is public and enabled in settings).

---

## Capabilities & Permissions

In compliance with OpenIDEA's extension security model, this plugin requests the following capabilities in `plugin.toml`:

| Capability | Purpose |
| --- | --- |
| `project.read` | Reads the current project root path to detect project names and determine Git repository status. |
| `editor.read` | Reads the active file path to determine file names, language icons, and editing status. |
| `vcs.read` | Reads version control status to detect current Git branch. |
| `events` | Subscribes to IDE lifecycle events (`editor-focused`, `file-opened`, `vcs-changed`, `settings-changed`) for reactive presence updates. |
| `storage` | Stores activity timestamps, the user's settings snapshot (so Privacy Mode and other choices survive a host reclaim), and the last working Discord socket. |
| `ipc:*` | Sandboxed local socket access to connect to the Discord desktop client (`discord-ipc-N`). OpenIDEA classifies `ipc:` as High risk (it is local socket access), but **no `process:` execution capability is requested**. |

---

## Idle & AFK behavior

OpenIDEA reclaims a plugin instance after 5 minutes with no events (its `deactivate` runs and the Discord IPC connection closes). `on-schedule:60` only activates a plugin that is already *Inactive* — the SDK states it "does not keep it alive" — so an active instance cannot rely on a periodic heartbeat.

Consequences:

- OpenIDEA closes the Discord IPC connection when it reclaims the instance. The plugin does **not** clear the Discord activity on deactivate and reconnects on activation (and on every subsequent event), so returning from AFK restores presence without a manual reconnect. The status bar only shows `Disconnected` when the Discord desktop client itself is unreachable.
- The plugin derives idle state from a **persisted** last-activity timestamp rather than a background ticker, and the configured `Inactivity Timeout` is evaluated whenever the IDE delivers an event. Reactivation does not reset the AFK clock.
- Privacy Mode and every other setting are persisted in host `storage`. On reactivation the plugin restores that snapshot without querying the host, because `activate()` has no project context and would otherwise read global defaults instead of your project-scoped choices. Until the first editor event refreshes settings with full project context, activation strictly enforces Safe Mode in memory so that stale public snapshots cannot leak presence.

---

## Configuration

Access settings anytime in OpenIDEA under **Settings › Plugins › Discord Rich Presence**:

- **General:**
  - `Enable Rich Presence`: Global master toggle.
  - `Detect Inactivity (AFK)`: Automatically updates presence when away from the keyboard.
  - `Inactivity Timeout (Minutes)`: Minutes before entering idle state (default: `5`).
  - `Status When Inactive`: Choose between `Show Idling`, `Keep Current Status`, or `Hide Presence`.
  - `Reset Timer on Return`: Reset session timer upon returning from idle.
- **Presence Content:**
  - `Main Line (File Active)`: Choose `{project} ({branch})`, `{project}`, `{file}`, or empty.
  - `Secondary Line (File Active)`: Choose `Editing {file}`, `{file}`, `{project}`, or empty.
  - `Prefix Files with 'Editing'`: Toggle prepending "Editing " to file names.
  - `Elapsed Time Counter`: Display `Project Time`, `File Open Time`, or `None`.
- **Appearance & Themes:**
  - `IDE Badge Icon Theme`: Theme for the OpenIDEA badge icon.
  - `File Language Icon Theme`: Theme for programming languages and file extensions.
- **Privacy & Buttons:**
  - `Privacy Mode`: Obfuscates file names and projects entirely.
  - `Show 'View Repository' Button`: Links your Discord status to your GitHub/GitLab repository.
  - `Show 'Get OpenIDEA' Button`: Adds a button linking to OpenIDEA.

---

## Commands & Shortcuts

OpenIDEA Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`):

- **Discord RPC: Toggle Presence** (`discord-rpc.toggle`): Pause or resume presence instantly (also accessible by clicking the status bar item).
- **Discord RPC: Reconnect** (`discord-rpc.reconnect`): Re-establishes connection with the Discord desktop client.
- **Discord RPC: About** (`discord-rpc.install-help`): Displays plugin information and link to documentation.

---

## Issues & Contributing

Found a bug, having connection issues, or want support for a new programming language or icon theme?

- **Bug Reports:** Open an issue using our [Bug Report template](https://github.com/MathCunha16/OpenIDEA-Discord-RCP/issues/new?template=bug_report.yml).
- **Feature Requests:** Suggest new languages, themes, or capabilities via the [Feature Request template](https://github.com/MathCunha16/OpenIDEA-Discord-RCP/issues/new?template=feature_request.yml).
- **Contributions:** Check out our [Contributing Guidelines](CONTRIBUTING.md) to learn how to build, test, and submit pull requests.

---

## Acknowledgements

- Icon themes and language definition mappings inspired by the community work in the [JetBrains Discord Integration](https://github.com/Azn9/JetBrains-Discord-Integration) ecosystem.

---

## License

This project is licensed under the [Apache License 2.0](LICENSE).

---

<div align="center">

Made with ❤️ by [**Matheus Cunha**](https://github.com/MathCunha16)

⭐ **If you find this project useful, please consider leaving a star on GitHub!**  
Your support helps keep the project active, motivates continued maintenance, and ensures new features and language icons keep coming!

</div>
