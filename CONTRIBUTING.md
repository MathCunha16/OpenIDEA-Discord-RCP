# Contributing to OpenIDEA Discord Rich Presence

Thank you for your interest in contributing to OpenIDEA Discord Rich Presence! This document explains how the project is organized, how to build and test it locally, and the rules for contributions.

---

## 1. Architecture Overview

Starting with **v0.3.0**, OpenIDEA Discord Rich Presence runs entirely as an in-process **WebAssembly Component Model** guest plugin (`wasip2`) compiled directly from Go:

```
OpenIDEA Host (IDE)
  └─► WebAssembly Component (wasm/gen/app -> dist/plugin.wasm)
        │
        ▼ Local IPC (\pipe\discord-ipc-0 or /run/user/.../discord-ipc-0)
      Discord Desktop Client
```

- **Pure WebAssembly (`dist/plugin.wasm`)**: Built using [`componentize-go`](https://github.com/bytecodealliance/componentize-go) and WIT bindings generated with [`wit-bindgen-go`](https://github.com/bytecodealliance/wit-bindgen-go).
- **Direct Local IPC**: Rather than relying on an external sidecar daemon, the WASM guest connects directly to the local Discord desktop client through sandboxed IPC socket capabilities granted in `plugin.toml`.
- **Zero Background Daemons**: No companion daemon installation, `$PATH` configuration, or external binary dependencies are needed.

---

## 2. Project Structure

```
.
├── assets/                  # Logos and packaged theme assets
│   ├── icons/themes/        # Icon themes (material, classic, atom, etc.)
│   └── logo.png             # Project banner logo
├── dist/                    # Output directory for compiled plugin.wasm
├── pkg/                     # Pure Go core logic packages
│   ├── config/              # Settings decoding, defaults, and presets
│   ├── discord/             # Discord IPC protocol, framing, and socket dialers
│   ├── git/                 # Git branch detection and HTTPS repo normalization
│   ├── ide/                 # Language detection database (475+ exts) and privacy sanitizer
│   ├── template/            # Presence text formatting engine and CDN icon resolver
│   └── tracker/             # AFK detection and elapsed time counters
├── wasm/                    # WebAssembly Component Model integration
│   └── gen/                 # Generated WIT bindings and component lifecycle (app)
├── wit/                     # WebAssembly Interface Type (WIT) world specifications
├── schemas/                 # JSON Schema defining native IDE settings UI
├── Makefile                 # Build, test, versioning, and packaging automation
├── plugin.toml              # OpenIDEA plugin manifest (^1.4)
├── README.md                # User-facing documentation and capability rationale
└── CONTRIBUTING.md          # Developer and contributor guidelines
```

---

## 3. Prerequisites

To build, test, and package the project locally:

- **Go**: 1.26 or higher (as specified in `go.mod`).
- **componentize-go**:
  ```sh
  go install github.com/bytecodealliance/componentize-go@latest
  ```
- **openidea-plugin CLI**:
  ```sh
  mkdir -p ~/.local/bin
  curl -sSL "https://github.com/l0ng-ai/openidea/releases/download/v0.1.40/openidea-plugin-linux-x86_64.gz" | gzip -d > ~/.local/bin/openidea-plugin
  chmod +x ~/.local/bin/openidea-plugin
  ```
- **Make** & **Zip**: For build automation and packaging.
- **OpenIDEA**: Installed locally for end-to-end testing.

---

## 4. Building the Project

The project provides standard `make` targets:

```sh
# Run all automated unit and race tests
make test

# Compile the WASM Component Model plugin (dist/plugin.wasm)
make wasm

# Package WASM and assets into cunha.discord-rpc-<VERSION>.oiplugin
make package

# Bump the version everywhere (Makefile and plugin.toml)
make bump VERSION=x.y.z

# Clean compiled outputs
make clean
```

`Makefile` `VERSION` is the single source of truth. Running `make bump VERSION=x.y.z` synchronizes `plugin.toml` automatically. The CI/release workflows enforce that Git tags strictly match `Makefile` and `plugin.toml`.

---

## 5. Testing

### Automated Unit and Race Tests
Run Go tests across all core packages and the WASM app package:
```sh
make test
```
Or run individually:
```sh
go test -count=1 -v -race ./...
cd wasm/gen && go test -count=1 -v -race ./app/...
```

### Static Analysis & Cross-Platform Checks
```sh
gofmt -l pkg wasm/gen/app
go vet ./...
(cd wasm/gen && go vet -unsafeptr=false ./app)
GOOS=windows go vet ./...
GOOS=darwin go vet ./...
```

### Manifest & Package Validation
```sh
openidea-plugin check .
openidea-plugin check cunha.discord-rpc-*.oiplugin
```

### Manual IDE Testing
1. Package the `.oiplugin` bundle:
   ```sh
   make package
   ```
2. Install the `.oiplugin` bundle in OpenIDEA (`Settings › Plugins › Install Plugin from Disk...`).
3. Verify:
   - Discord status updates when opening/switching files or focusing editor tabs.
   - Status bar widget (`● Discord`) reflects status and toggles pause on click.
   - Settings in `Settings › Plugins › Discord Rich Presence` apply reactively.

---

## 6. Language & Icon Parity

When adding support for a new programming language or file extension:

1. Update `pkg/ide/languages.go` (or `languages_data.go`).
2. For multi-part file extensions (e.g. `.custom.config`), ensure composite extension matching is handled cleanly.
3. Verify that the asset identifier exists in the supported icon themes under `assets/icons/themes/`.
4. Add a test case to `TestDetectLanguage` in [pkg/ide/ide_test.go](file:///home/cunha/Documentos/Programacao/OpenIDEA-Discord-Rcp/pkg/ide/ide_test.go).

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
- **`style`**: Changes that do not affect the meaning of the code (formatting, whitespace, etc.).

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
4. **Run Tests:** Ensure tests and validations pass:
   ```sh
   make test
   make wasm
   openidea-plugin check .
   ```
5. **Commit Your Changes:** Adhere to the Conventional Commits format described above:
   ```sh
   git commit -m "feat(language): add support for Gleam files (.gleam)"
   ```
6. **Push to Your Fork:**
   ```sh
   git push origin feat/your-feature-name
   ```
7. **Open a Pull Request:** Target the `main` branch and ensure all CI checks pass.
