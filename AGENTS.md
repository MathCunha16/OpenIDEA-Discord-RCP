# AGENTS.md

OpenIDEA Discord Rich Presence plugin.
Pure **WebAssembly Component Model** architecture (`wasip2`) compiled directly from Go using `componentize-go` and WIT bindings.
Connects directly to the local Discord desktop client through sandboxed IPC sockets (`ipc:...`), with **zero companion daemons or background processes**.

Read `CONTRIBUTING.md` for the contributor workflow. `plugin.toml` is the OpenIDEA manifest; `schemas/settings.json` drives the settings UI.

## Commands

- `make test`: Go unit and race tests. Runs `go test -count=1 -v -race ./...` and `cd wasm/gen && go test -count=1 -v -race ./app/...`.
- `make wasm`: builds `dist/plugin.wasm` via `componentize-go`.
- `make package`: packages `cunha.discord-rpc-<VERSION>.oiplugin` (needs `zip`).
- `make clean`: removes `dist/` and `*.oiplugin`.
- Formatting check: `gofmt -l pkg wasm/gen/app`.
- Cross-platform check: `GOOS=windows go vet ./...` and `GOOS=darwin go vet ./...`. (For `wasm/gen`, use `(cd wasm/gen && go vet -unsafeptr=false ./app)` due to `wit-bindgen-go` pointer arithmetic).
- Go version comes from `go.mod` (1.26).
- Manifest check: `openidea-plugin check .` and `openidea-plugin check cunha.discord-rpc-*.oiplugin`.
- CI/Release: `.github/workflows/ci.yml` runs formatting, tests, vet, and manifest validation on PRs and `main`; `.github/workflows/release.yml` builds and publishes the `.oiplugin` and `checksums.txt` on `v*` tags.

## Version bumps

- `Makefile` `VERSION` is the single source of truth (also sets the `.oiplugin` filename).
- Run `make bump VERSION=x.y.z`: rewrites `VERSION :=` in `Makefile` and synchronizes `plugin.toml` (`version` and numeric `change-notes` prefix).
- Update the change notes text in `plugin.toml`.
- `.github/workflows/release.yml` validates that the Git tag strictly matches `plugin.toml` and `Makefile` (`grep '^VERSION :=' Makefile`).

## Runtime behavior

- OpenIDEA loads `dist/plugin.wasm` in its Wasmtime sandbox and interacts with it via the WIT `openidea:plugin/app` world.
- The component runs inside the IDE process and connects directly to Discord's local IPC socket or named pipe using granted `ipc:...` capabilities.
- Status bar item (`discord-rpc-status`) dynamically reflects current state:
  - `● Discord` (Active & connected)
  - `○ Discord (Paused)` (User manually paused presence)
  - `✕ Discord (Disconnected)` (Discord desktop app not running or socket unreachable)
  - `◌ Discord (Off)` (Presence disabled in plugin settings)
- Events handled reactively: `editor-focused`, `file-opened`, `vcs-changed`, `settings-changed`, `scheduled`, and commands (`discord-rpc.toggle`, `discord-rpc.reconnect`, `discord-rpc.install-help`).
- Debouncing & Throttling: Activity updates are throttled to comply with Discord's IPC rate limits while maintaining snappy editor responsiveness.

## Privacy invariants

- Ignored inputs: Anything under a `.openidea` directory, `settings.json`/`settings.toml`/`settings` inside an `openidea` config folder, and virtual `{...}` documents (e.g. `{diff}`, `{terminal}`). Enforced by `ide.IsIgnoredFile` in `pkg/ide/privacy.go` and `wasm/gen/app/app.go`. Ignored files do not trigger presence updates or reset the AFK clock.
- Privacy Mode: When enabled, obscures file and project names with generic descriptions.
- Sensitive File Protection: Redacts passwords, certificates, private keys, `.env` files, and credentials.
- Git Remote Sanitization: Strips usernames, passwords, query parameters, and fragments from remote URLs before creating repository links.
- No Outbound Network Traffic: The plugin makes zero HTTP or external network calls; all traffic is strictly local IPC.

## Code map

- `pkg/config/`: Settings decoding, defaults, and label preset normalization.
- `pkg/discord/`: Discord IPC protocol, handshake, framing, and cross-platform socket/pipe dialers (`socket_unix.go`, `socket_windows.go`).
- `pkg/git/`: Git branch detection and HTTPS remote repository URL sanitization.
- `pkg/ide/`: Language recognition database (250+ languages, 475+ extensions), privacy sanitizer, and ignored files filter.
- `pkg/template/`: Rich Presence text template rendering and Discord CDN icon resolver.
- `pkg/tracker/`: Inactivity / AFK detection and elapsed session timers.
- `wasm/gen/`: WIT bindings generated with `wit-bindgen-go`.
- `wasm/gen/app/`: WebAssembly Component implementation, event loop, state management, and status bar updates.
- `wit/`: WebAssembly Interface Type (WIT) specifications describing the OpenIDEA plugin world.

## Generated and data files

- `pkg/template/theme_assets_generated.go` and `pkg/ide/languages_data.go` contain icon and language databases.
- Adding a language or extension: Update `pkg/ide/languages.go` (or `languages_data.go`), add a test case to `TestDetectLanguage` in `pkg/ide/ide_test.go`, and verify the icon exists under `assets/icons/themes/<theme>/`.

## Conventions

- Conventional Commits, lowercase, imperative, English: `feat(scope): add ...`, `fix(scope): fix ...`, `chore(release): ...`.
- Do not commit `bin/`, `dist/`, `*.oiplugin`, or `target/` (gitignored). `skills/` and `.openidea/` are local tool state.
