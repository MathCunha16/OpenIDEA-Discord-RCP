# Contributing to OpenIDEA Discord Rich Presence

Thank you for your interest in contributing to OpenIDEA Discord Rich Presence! This document explains how the project is organized, how to build and test it locally, and the rules for contributions.

---

## 1. Architecture Overview

The plugin follows a hybrid architecture designed for performance, security, and low memory consumption:

```
OpenIDEA Host (IDE)
  └─► WASM Bridge (Rust Component Model / crates/wasm-bridge)
        │
        │ host::start child process, JSON events on stdin
        ▼
      Native Sidecar Daemon (Go / cmd/daemon)
        │
        ▼ Local IPC (\pipe\discord-ipc-0 or /run/user/.../discord-ipc-0)
      Discord Desktop Client
```

1. **WASM Bridge (`crates/wasm-bridge`)**: Runs inside OpenIDEA's sandboxed Wasmtime runtime. It listens for editor focus, file opening, VCS changes, and settings changes, relaying these events to the native daemon.
2. **Native Sidecar Daemon (`cmd/daemon`)**: A standalone, ultra-low-footprint Go binary (~5 MB RSS). It reads newline-delimited JSON events from stdin and exits when stdin closes. It manages Unix/Named-Pipe IPC with the Discord client, keeps track of idle/AFK timeouts, sanitizes sensitive files, and formats rich presence payloads.

---

## 2. Project Structure

```
.
├── assets/                  # Logos and packaged theme assets
│   ├── icons/themes/        # Icon themes (material, classic, atom, etc.)
│   └── logo.png             # Project banner logo
├── bin/                     # Output directory for compiled native daemon
├── cmd/daemon/              # Native sidecar daemon (Go main package)
├── crates/wasm-bridge/      # WASM Component Model guest bridge (Rust)
├── dist/                    # Output directory for compiled plugin.wasm
├── internal/                # Internal Go packages for the daemon
│   ├── config/              # JSON settings unmarshaling and validation
│   ├── discord/             # Discord IPC client, socket dialers, framing
│   ├── git/                 # Git branch detection and HTTPS repo normalization
│   ├── ide/                 # Language detection database (475+ exts) and privacy sanitizer
│   ├── template/            # Presence text formatting engine and CDN resolver
│   └── tracker/             # AFK detection and elapsed time counters
├── schemas/                 # JSON Schema defining native IDE settings UI
├── Makefile                 # Build, test, and packaging automation
├── plugin.toml              # OpenIDEA plugin manifest (^1.4)
├── README.md                # User-facing documentation and capability rationale
└── CONTRIBUTING.md          # Developer and contributor guidelines
```

---

## 3. Prerequisites

To build and test the entire project, ensure you have the following installed:

- **Go**: 1.22 or higher.
- **Rust**: Latest stable with the `wasm32-wasip2` target:
  ```sh
  rustup target add wasm32-wasip2
  ```
- **Make** & **Zip**: For package bundling.
- **OpenIDEA**: Installed locally for testing plugin installation.

---

## 4. Building the Project

The project includes convenient `make` targets:

```sh
# Build native daemon for current host (bin/openidea-discord-rpc-daemon)
make daemon

# Cross-compile native daemon for all supported platforms (Linux, macOS, Windows)
make cross-compile

# Generate SHA-256 checksums in bin/dist/checksums.txt
make checksums

# Build WASM bridge component only (dist/plugin.wasm)
make wasm

# Run all automated unit and race tests
make test

# Package WASM and metadata into cunha.discord-rpc-<VERSION>.oiplugin
make package
```

---

## 5. Testing

### Go Automated Tests
All packages include comprehensive unit and race detection tests:
```sh
go test -count=1 -v -race ./...
```

### Rust WASM Tests
```sh
cargo test --manifest-path crates/wasm-bridge/Cargo.toml
```

### Manual IDE Testing
1. Build the `.oiplugin` package:
   ```sh
   make package
   ```
2. Install the generated `.oiplugin` file into OpenIDEA (`Settings › Plugins › Install Plugin from Disk...`).
3. Verify:
   - Discord status updates when opening/switching files.
   - Status bar widget (`● Discord`) toggles presence.
   - Settings changes in `Settings › Plugins › Discord Rich Presence` apply reactively.

---

## 6. Language & Icon Parity

When adding support for a new programming language or file extension:

1. Update `internal/ide/languages.go` or `internal/ide/languages_data.go`.
2. For multi-part file extensions (e.g. `.custom.config`), ensure composite extension matching is handled cleanly.
3. Verify that the asset identifier exists in the supported icon themes under `assets/icons/themes/`.
4. Add a test case to `TestDetectLanguage` in [internal/ide/ide_test.go](file:///home/cunha/Documentos/Programacao/OpenIDEA-Discord-Rcp/internal/ide/ide_test.go).

---

## 7. Commit Message Convention

This project strictly adheres to the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```text
<type>(<scope>): <short description>
```

### Allowed Types
- **`feat`**: A new feature or capability.
- **`fix`**: A bug fix.
- **`refactor`**: Code changes that neither fix a bug nor add a feature.
- **`chore`**: Maintenance, build tool changes, dependency bumps.
- **`docs`**: Documentation updates only.
- **`test`**: Adding or updating tests.
- **`perf`**: Performance optimizations.
- **`style`**: Changes that do not affect the meaning of the code (formatting, whitespace, icon viewBox, etc.).

### Guidelines
- Write all commit messages, PR titles, and code comments in **English**.
- Use lowercase for the type and description.
- Use imperative, present tense (e.g., `add ...`, `fix ...`, not `added` or `fixes`).

---

## 8. Contribution & Pull Request Workflow

Follow these steps to submit your contributions:

1. **Fork the Repository:** Create your own fork on GitHub.
2. **Create a Feature Branch:** Always branch off `main` with a descriptive name:
   ```sh
   git checkout -b feat/your-feature-name
   # or for bug fixes:
   git checkout -b fix/issue-description
   ```
3. **Make Your Changes:** Follow the project structure, documentation rules, and code cleanliness standards.
4. **Run Tests:** Ensure both Go and Rust suites pass without errors or race warnings:
   ```sh
   make test
   cargo test --manifest-path crates/wasm-bridge/Cargo.toml
   ```
5. **Commit Your Changes:** Adhere to the Conventional Commits format described above:
   ```sh
   git commit -m "feat(language): add support for Gleam files (.gleam)"
   ```
6. **Push to Your Fork:**
   ```sh
   git push origin feat/your-feature-name
   ```
7. **Open a Pull Request:**
   - Target the `main` branch.
   - Describe what the PR accomplishes, linking any relevant issues.
   - Ensure the CI checks pass. Maintainers will review your PR and provide feedback promptly!
