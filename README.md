<p align="center">
  <img src="assets/logo.png" alt="OpenIDEA Discord Rich Presence" width="200"/>
</p>
<h1 align="center">OpenIDEA Discord Rich Presence</h1>

<p align="center">
  A fast, lightweight, and privacy-conscious Discord Rich Presence plugin for <a href="https://github.com/l0ng-ai/openidea">OpenIDEA</a>. Automatically displays your active project, file details, language icons, Git branch, and elapsed session timer on your Discord profile.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/OpenIDEA-%5E1.4-orange" alt="OpenIDEA Engine"/>
  <img src="https://img.shields.io/badge/license-Apache--2.0-blue.svg" alt="License"/>
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go" alt="Go"/>
  <img src="https://img.shields.io/badge/Rust-Wasm-black?logo=rust" alt="Rust Wasm"/>
</p>

---

## Features

- **Event-Driven & Instant Sync:** Updates presence seamlessly on editor focus, file navigation, VCS branch changes, and IDE settings adjustments.
- **Ultra-Lightweight Footprint:** Native sidecar daemon consumes only **~5 MB of RAM** with **0.0% idle CPU**.
- **Exhaustive Language & File Recognition:** Parity with 250+ language definitions covering over **475 file extensions** (including composite extensions like `.tar.gz`, `.ng.html`, `.component.ts`, `.spec.js`) and **105 exact file manifests** (`Dockerfile`, `Cargo.toml`, `Makefile`, etc.).
- **6 Rich Icon Themes:**
  - **Material** (default)
  - **Classic**
  - **Atom**
  - **Catppuccin Mocha**
  - **Catppuccin Latte**
  - **Flat**
- **Interactive Status Bar Widget:** Toggle presence with 1-click directly from OpenIDEA's bottom bar (`● Discord` / `○ Discord (Paused)`).
- **Customizable Presence Content:** Choose custom line formats for active files and projects, toggle "Editing " prefixes, and pick between project time, file open time, or no timer.
- **Inactivity & AFK Detection:** Automatically transitions presence to idle or hides presence after a configurable inactivity threshold.
- **Privacy & Security First:**
  - **Privacy Mode:** Completely hides file, project, and branch names from Discord with a single toggle.
  - **Sensitive File Protection:** Automatic sanitization of `.env`, private keys, secrets, and credential files.
  - **Optional Buttons:** Choose whether to expose your public Git repository or IDE download link on Discord.

---

## Daemon Installation

Because OpenIDEA executes plugins within a strictly isolated WebAssembly sandbox (`wasm32-wasip2`), guest plugins cannot directly access native OS IPC sockets (such as Discord's local IPC). To bridge this, this plugin uses a lightweight companion daemon (`openidea-discord-rpc-daemon`) that runs natively on your system and is discovered via your system `$PATH`.

### 1. Install the Daemon

Precompiled, reproducible binaries built from Git tags with SHA-256 checksums are attached to every [GitHub Release](https://github.com/MathCunha16/OpenIDEA-Discord-Rcp/releases).

#### Linux & macOS (Quick Install)
Run this command in your terminal to download the latest daemon for your platform into `~/.local/bin` (or `/usr/local/bin`):

```bash
# Ensure ~/.local/bin exists and is in your PATH
mkdir -p ~/.local/bin

# Linux x86_64
curl -sSL "https://github.com/MathCunha16/OpenIDEA-Discord-Rcp/releases/latest/download/openidea-discord-rpc-daemon-linux-amd64" -o ~/.local/bin/openidea-discord-rpc-daemon && chmod +x ~/.local/bin/openidea-discord-rpc-daemon

# Linux ARM64
# curl -sSL "https://github.com/MathCunha16/OpenIDEA-Discord-Rcp/releases/latest/download/openidea-discord-rpc-daemon-linux-arm64" -o ~/.local/bin/openidea-discord-rpc-daemon && chmod +x ~/.local/bin/openidea-discord-rpc-daemon

# macOS Apple Silicon (M1/M2/M3/M4)
# curl -sSL "https://github.com/MathCunha16/OpenIDEA-Discord-Rcp/releases/latest/download/openidea-discord-rpc-daemon-darwin-arm64" -o ~/.local/bin/openidea-discord-rpc-daemon && chmod +x ~/.local/bin/openidea-discord-rpc-daemon

# macOS Intel
# curl -sSL "https://github.com/MathCunha16/OpenIDEA-Discord-Rcp/releases/latest/download/openidea-discord-rpc-daemon-darwin-amd64" -o ~/.local/bin/openidea-discord-rpc-daemon && chmod +x ~/.local/bin/openidea-discord-rpc-daemon
```

#### Windows (PowerShell)
Download `openidea-discord-rpc-daemon-windows-amd64.exe` and place it in any directory included in your user `Path` environment variable (for example, `%USERPROFILE%\bin`):

```powershell
New-Item -ItemType Directory -Force -Path "$HOME\bin"
Invoke-WebRequest -Uri "https://github.com/MathCunha16/OpenIDEA-Discord-Rcp/releases/latest/download/openidea-discord-rpc-daemon-windows-amd64.exe" -OutFile "$HOME\bin\openidea-discord-rpc-daemon.exe"

# If $HOME\bin is not yet in your PATH, add it:
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$HOME\bin", [EnvironmentVariableTarget]::User)
```

> [!TIP]
> **Updating the daemon:** Simply replace the existing binary in your `$PATH` with the newer release and trigger **Discord RPC: Reconnect** (`discord-rpc.reconnect`) from OpenIDEA's Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`).

---

## Supported Platforms & IPC Transports

| Platform | Architecture | Binary Name | Discord IPC Transport |
| --- | --- | --- | --- |
| **Linux** | `x86_64` (amd64) | `openidea-discord-rpc-daemon-linux-amd64` | Unix domain sockets (`$XDG_RUNTIME_DIR/discord-ipc-N` & Flatpak portals) |
| **Linux** | `aarch64` (arm64) | `openidea-discord-rpc-daemon-linux-arm64` | Unix domain sockets (`$XDG_RUNTIME_DIR/discord-ipc-N` & Flatpak portals) |
| **macOS** | Apple Silicon (`arm64`) | `openidea-discord-rpc-daemon-darwin-arm64` | Unix domain sockets (`$TMPDIR/discord-ipc-N` or `/tmp/discord-ipc-N`) |
| **macOS** | Intel (`x86_64`) | `openidea-discord-rpc-daemon-darwin-amd64` | Unix domain sockets (`$TMPDIR/discord-ipc-N` or `/tmp/discord-ipc-N`) |
| **Windows** | `x86_64` (amd64) | `openidea-discord-rpc-daemon-windows-amd64.exe` | Windows Named Pipes (`\\.\pipe\discord-ipc-N`) |

---

## Data Flow & Privacy Disclosure

Transparency is paramount. Here is exactly what data is exchanged across the architecture:

```
[OpenIDEA Editor]
       │
       ▼ (1) IDE Events (Active file path, project root, git branch, settings)
[WASM Bridge]
       │
       ▼ (2) Process Invocation (`openidea-discord-rpc-daemon --topic ... --payload ...`)
[Native Daemon] ── (3) Sanitization & Privacy Filter
       │
       ▼ (4) Discord RPC Handshake & SET_ACTIVITY payload
[Discord Client] (Local IPC Socket)
```

1. **What the WASM Bridge sends to the Daemon:**
   - Event topic (e.g. `editor-focused`, `file-opened`, `vcs-changed`).
   - Project directory path (to identify project root & Git repository).
   - Active file relative/absolute path.
   - User settings JSON (theme, privacy preferences, custom line templates).

2. **What the Daemon sanitizes:**
   - **Privacy Mode:** When enabled, the daemon replaces project and file names with generic descriptions (`Working on a project`).
   - **Sensitive File Filter:** Even with Privacy Mode disabled, the daemon automatically redacts passwords, certificates, private keys, `.env` files, and secrets.
   - **No Outbound Network Access:** The daemon binds strictly to local IPC and makes zero outbound HTTP/network requests.

3. **What is sent to Discord:**
   - Details text (e.g. `Project Name (main)`).
   - State text (e.g. `Editing main.rs`).
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
| `events` | Subscribes to IDE lifecycle events (`editor-focused`, `file-opened`, `vcs-changed`, `settings-changed`) for reactive presence updates. |
| `process:openidea-discord-rpc-daemon` | **High-Risk Process Execution:** Launches the native companion daemon installed in the user's `$PATH`. The daemon connects to the local Discord desktop client via local IPC sockets, tracks AFK idle states, and formats Rich Presence payloads. The daemon makes **no outbound network requests**. |

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
- **Discord RPC: Reconnect** (`discord-rpc.reconnect`): Force reconnection to Discord IPC socket and verify daemon health.
- **Discord RPC: Setup Guide** (`discord-rpc.install-help`): Opens the daemon setup instructions and release download page in your browser.

---

## Issues & Contributing

Found a bug, having connection issues, or want support for a new programming language or icon theme?

- **Bug Reports:** Open an issue using our [Bug Report template](https://github.com/MathCunha16/OpenIDEA-Discord-RCP/issues/new?template=bug_report.yml) (please include your daemon logs).
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

