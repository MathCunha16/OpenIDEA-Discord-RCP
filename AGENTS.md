# AGENTS.md

OpenIDEA Discord Rich Presence plugin. Two halves that must stay in sync:
- **WASM bridge** `crates/wasm-bridge/` (Rust, `wasm32-wasip2`): runs inside OpenIDEA's sandbox. Cannot open Discord IPC sockets, so it shells out to the native daemon.
- **Native daemon** `cmd/daemon/` + `internal/` (Go): the `openidea-discord-rpc-daemon` binary, found by the bridge via `$PATH`. Owns the Discord IPC connection, AFK tracking, sanitizing, and payload formatting.

Read `CONTRIBUTING.md` for the workflow. `plugin.toml` is the OpenIDEA manifest; `schemas/settings.json` drives the settings UI.

## Commands

- `make test`: Go tests with `-race` (passes). Single package or test: `go test -count=1 -race ./internal/ide -run TestDetectLanguage`.
- `cargo test --manifest-path crates/wasm-bridge/Cargo.toml`: runs on the host target. Do not add `--target`.
- `make daemon`: host daemon to `bin/openidea-discord-rpc-daemon` (`CGO_ENABLED=0`).
- `make wasm`: builds `dist/plugin.wasm`. Needs `rustup target add wasm32-wasip2` and network access (git dependency `openidea-plugin` pinned to a tag).
- `make package`: builds `cunha.discord-rpc-<VERSION>.oiplugin` (needs `zip`).
- `make clean`: removes `bin/`, `dist/`, `*.oiplugin`.
- Cross-platform check after touching OS-specific code: `GOOS=windows go vet ./...` and `GOOS=darwin go vet ./...`. OS-specific files: `internal/discord/{dialer,socket}_{unix,windows}.go`, `cmd/daemon/instance_lock_{unix,windows}.go`.
- Go version comes from `go.mod` (1.26). README/CONTRIBUTING say "1.22+", which is stale.
- There are no CI workflows in the repo (`.github/` only has issue templates). Run `make test` and the cargo tests locally before finishing.

## Version bumps: keep these in sync

A plugin release touches all of these:
- `Makefile` `VERSION` (also sets the `.oiplugin` filename)
- `plugin.toml` `version` (plus `change-notes`)
- `crates/wasm-bridge/Cargo.toml` `version`
- `crates/wasm-bridge/src/lib.rs` `MIN_DAEMON_VERSION`. The bridge compares the daemon's `--version` output to this and shows "Daemon Outdated" if it is lower.

The daemon's version comes from `-ldflags "-X main.DaemonVersion=..."`. A plain `go build` reports `dev`, which does not parse as a version and so is never flagged as outdated.

## Runtime behavior

- The bridge starts `openidea-discord-rpc-daemon` once per plugin instance with `host::start` and writes one JSON event per line to its stdin. Closing stdin (IDE quit, plugin disabled, `deactivate`) makes the daemon clear the Discord activity and exit. `deactivate` also closes stdin, waits briefly, then kills the child.
- Stdout and stderr of the child are kept in memory by the host until polled. The bridge drains them with `poll` on every event batch. The daemon logs only to `$TMPDIR/openidea-discord-rpc.log`.
- Single instance: the daemon takes an exclusive lock on `$TMPDIR/openidea-discord-rpc-<uid>.lock` (flock on Unix, LockFileEx on Windows). A second instance exits with code 3, and the bridge retries on the next heartbeat.
- Version gate: `--version` is probed with `run_process` before a child is started. An outdated or missing daemon is never started.
- Heartbeat: `on-schedule:60` sends a `scheduled` event that carries the full settings JSON. It also restarts a dead child.
- The `--topic`, `--payload`, `--file`, `--project`, `--shutdown` and `--reconnect` CLI flags no longer exist. Only `--version` is accepted.
- Full end-to-end testing needs OpenIDEA and the Discord desktop client running. Install the `.oiplugin` via `Settings › Plugins › Install Plugin from Disk`. The bridge only finds the daemon on `$PATH`, so `bin/` must be on `PATH` for local testing.
- Discord IPC transports: Unix sockets on Linux/macOS, named pipes `\\.\pipe\discord-ipc-N` on Windows.

## Privacy invariants (keep both layers)

- The bridge enforces privacy mode before file/project paths reach the daemon's stdin (`handle_events` in `lib.rs`). The daemon also sanitizes in `internal/ide` and `internal/template`. Do not remove either layer.
- Ignored inputs: `settings.json`, `settings.toml`, anything under a `.openidea` directory, and virtual `{...}` documents.
- The daemon must make no outbound network requests (advertised in README). Do not add HTTP calls to the daemon.

## Settings wire format

- The bridge sends camelCase JSON from `BridgeSettings` in `lib.rs` (`detectIdle`, `whileIdle` with labels like `"Show Idling"`).
- `internal/config/settings.go` `UnmarshalJSON` accepts both camelCase and snake_case, and `NormalizePresets` maps UI labels to internal values. Both were verified to decode the bridge's keys. When adding a setting, update `BridgeSettings`, `schemas/settings.json`, and the Go `RawSettings` aliases together.
- Settings reach the daemon on `plugin-activated`, on toggle/`settings-changed`, and on every `scheduled` heartbeat (the heartbeat carries the full settings JSON). A respawned daemon starts with defaults until the next heartbeat.

## Process permission is pinned by hash

- OpenIDEA pins `process:openidea-discord-rpc-daemon` to the binary's blake3 hash at approval time. Replacing the binary yields a `Denied` error ("changed since it was approved"). Installing it after the permission was granted yields a `Failed` error ("not found on PATH when this permission was allowed").
- Updating or first-installing the daemon therefore needs the user to clear and re-approve the permission in `Settings › Plugins`. Reconnect alone does not work.

## Code map

- `cmd/daemon/main.go`: CLI (`--version`), stdin event loop, coordinator (debounce 1.5s, 2s publish throttle, 350ms settings debounce, 15s idle ticker).
- `cmd/daemon/instance_lock*.go`: per-user single-instance lock.
- `internal/discord/`: IPC client, framing, platform dialers.
- `internal/template/`: presence text rendering and CDN icon URLs.
- `internal/ide/`: event parsing, language detection, privacy sanitizer.
- `internal/tracker/`: idle detection and elapsed-time timers.
- `internal/git/`: branch detection and repo URL normalization.
- `internal/config/`: settings defaults and parsing.

## Generated and data files

- `internal/template/theme_assets_generated.go` and `internal/ide/languages_data.go` say "DO NOT EDIT" and "generated", but the repo has no generator. Do not hand-edit without knowing the source.
- Adding a language or extension: CONTRIBUTING §6. Edit `internal/ide/languages.go` (or the data file), add a case to `TestDetectLanguage` in `internal/ide/ide_test.go`, and confirm the icon exists under `assets/icons/themes/<theme>/`.

## Conventions

- Conventional Commits, lowercase, imperative, English: `feat(language): add ...`. Types are listed in CONTRIBUTING §7.
- Do not commit `bin/`, `dist/`, `*.oiplugin`, or `target/` (all gitignored). `skills/` and `.openidea/` are local tool state. Do not modify or commit them.
