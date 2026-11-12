use openidea_plugin::{
    Context, Error, Event, EventKind, Invocation, NotificationAction, NotificationLevel, Plugin,
    StatusItem, host, register_plugin,
};

const MIN_DAEMON_VERSION: &str = "0.1.1";
const STATUS_BAR_ID: &str = "discord-rpc-status";

#[derive(Clone, Copy, PartialEq, Eq, Debug, Default)]
enum DaemonStatus {
    #[default]
    Unknown,
    Connected,
    Missing,
    Outdated,
}

#[derive(Default)]
struct DiscordRpcBridge {
    paused: bool,
    daemon_status: DaemonStatus,
    installed_version: String,
    notified_state: Option<DaemonStatus>,
    last_sent: Option<(String, String, String)>,
}

fn parse_version(v: &str) -> Option<(u32, u32, u32)> {
    let clean = v.trim();
    // Look for a token containing digits and dots (handles "v0.1.1", "daemon 0.1.1-beta")
    for token in clean.split_whitespace() {
        let t = token.trim_start_matches('v');
        let semver = t.split('-').next().unwrap_or(t);
        let mut parts = semver.split('.');
        if let (Some(maj), Some(min)) = (parts.next(), parts.next()) {
            if let (Ok(major), Ok(minor)) = (maj.parse::<u32>(), min.parse::<u32>()) {
                let patch = parts.next().and_then(|p| p.parse::<u32>().ok()).unwrap_or(0);
                return Some((major, minor, patch));
            }
        }
    }
    None
}

fn is_version_outdated(installed: &str, required: &str) -> bool {
    match (parse_version(installed), parse_version(required)) {
        (Some(inst), Some(req)) => inst < req,
        _ => false,
    }
}

fn update_status_bar(enabled: bool, paused: bool, daemon_status: DaemonStatus, installed_version: &str) {
    let (text, tooltip, command) = match daemon_status {
        DaemonStatus::Missing => (
            "⚠ Discord (Daemon Missing)",
            "Discord RPC daemon not found on PATH or permission denied. Click to check again.".to_string(),
            "discord-rpc.reconnect",
        ),
        DaemonStatus::Outdated => (
            "↻ Discord (Daemon Outdated)",
            format!(
                "Discord RPC daemon is outdated (v{} installed, v{} required). Click to update.",
                if installed_version.is_empty() { "unknown" } else { installed_version },
                MIN_DAEMON_VERSION
            ),
            "discord-rpc.install-help",
        ),
        _ => {
            if !enabled {
                (
                    "○ Discord (Off)",
                    "Discord Rich Presence: Disabled in settings".to_string(),
                    "discord-rpc.toggle",
                )
            } else if paused {
                (
                    "○ Discord (Paused)",
                    "Discord Rich Presence: Paused (Click to resume)".to_string(),
                    "discord-rpc.toggle",
                )
            } else {
                (
                    "● Discord",
                    "Discord Rich Presence: Active (Click to pause)".to_string(),
                    "discord-rpc.toggle",
                )
            }
        }
    };

    host::set_status_item(
        STATUS_BAR_ID,
        &StatusItem {
            text: text.to_string(),
            tooltip: Some(tooltip),
            command: Some(command.to_string()),
        },
    );
}

#[derive(serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct BridgeSettings {
    enabled: bool,
    detect_idle: bool,
    idle_timeout_mins: i64,
    while_idle: String,
    reset_open_time_on_return: bool,
    file_first_line: String,
    file_second_line: String,
    prefix_reading_editing: bool,
    file_large_icon_role: String,
    file_small_icon_role: String,
    file_time_mode: String,
    project_first_line: String,
    project_second_line: String,
    project_time_mode: String,
    time_mode: String,
    application_theme: String,
    language_theme: String,
    privacy_mode: bool,
    show_git_button: bool,
    show_ide_button: bool,
    debounce_ms: i64,
    application_id: String,
}

fn load_bridge_settings() -> BridgeSettings {
    BridgeSettings {
        enabled: host::setting_bool("enabled").unwrap_or(true),
        detect_idle: host::setting_bool("detectIdle").unwrap_or(true),
        idle_timeout_mins: host::setting_number("idleTimeoutMins").map(|n| n as i64).unwrap_or(5),
        while_idle: host::setting_string("whileIdle").unwrap_or_else(|| "Show Idling".to_string()),
        reset_open_time_on_return: host::setting_bool("resetOpenTimeOnReturn").unwrap_or(false),
        file_first_line: host::setting_string("fileFirstLine").unwrap_or_else(|| "Project Name (Branch)".to_string()),
        file_second_line: host::setting_string("fileSecondLine").unwrap_or_else(|| "Editing file".to_string()),
        prefix_reading_editing: host::setting_bool("prefixReadingEditing").unwrap_or(true),
        file_large_icon_role: host::setting_string("fileLargeIconRole").unwrap_or_else(|| "File Language Icon".to_string()),
        file_small_icon_role: host::setting_string("fileSmallIconRole").unwrap_or_else(|| "Application (OpenIDEA) Icon".to_string()),
        file_time_mode: host::setting_string("fileTimeMode").unwrap_or_default(),
        project_first_line: host::setting_string("projectFirstLine").unwrap_or_else(|| "Project Name (Branch)".to_string()),
        project_second_line: host::setting_string("projectSecondLine").unwrap_or_else(|| "Navigating project".to_string()),
        project_time_mode: host::setting_string("projectTimeMode").unwrap_or_else(|| "Project Time".to_string()),
        time_mode: host::setting_string("timeMode").unwrap_or_else(|| "Project Time".to_string()),
        application_theme: host::setting_string("applicationTheme").unwrap_or_else(|| "Material".to_string()),
        language_theme: host::setting_string("languageTheme").unwrap_or_else(|| "Classic".to_string()),
        privacy_mode: host::setting_bool("privacyMode").unwrap_or(false),
        show_git_button: host::setting_bool("showGitButton").unwrap_or(true),
        show_ide_button: host::setting_bool("showIDEButton").unwrap_or(true),
        debounce_ms: host::setting_number("debounceMs").map(|n| n as i64).unwrap_or(1500),
        application_id: host::setting_string("applicationId").unwrap_or_else(|| "1569761142878896160".to_string()),
    }
}

fn current_settings_json(paused: bool) -> String {
    let mut settings = load_bridge_settings();
    settings.enabled = settings.enabled && !paused;
    serde_json::to_string(&settings).unwrap_or_default()
}

fn is_ignored_file(path: &str) -> bool {
    let lower = path.to_lowercase().replace('\\', "/");
    let file_name = lower.rsplit('/').next().unwrap_or(&lower);
    if file_name == "settings.json" || file_name == "settings.toml" || lower.starts_with('{') {
        return true;
    }
    // Only ignore if .openidea is an actual path component (directory)
    lower.split('/').any(|segment| segment == ".openidea")
}

impl DiscordRpcBridge {
    fn update_ui(&self) {
        let enabled = host::setting_bool("enabled").unwrap_or(true);
        update_status_bar(enabled, self.paused, self.daemon_status, &self.installed_version);
    }

    fn check_daemon_version(&mut self) -> Result<String, Error> {
        let output = host::run_process(
            "openidea-discord-rpc-daemon",
            &["--version".to_string()],
            None,
        )?;
        if output.status != Some(0) {
            return Ok(String::new());
        }
        let version_str = output.stdout.trim().to_string();
        Ok(version_str)
    }

    fn set_daemon_status(&mut self, new_status: DaemonStatus, version: Option<String>, err: Option<&Error>) {
        self.daemon_status = new_status;
        if let Some(v) = version {
            self.installed_version = v;
        }
        self.update_ui();

        if self.notified_state == Some(new_status) {
            return;
        }
        self.notified_state = Some(new_status);

        match new_status {
            DaemonStatus::Missing => {
                let action = NotificationAction {
                    label: "Setup Guide".to_string(),
                    command: "discord-rpc.install-help".to_string(),
                    argument: None,
                };
                let (title, body) = match err {
                    Some(Error::Denied(cap)) => (
                        "Discord RPC Permission Denied",
                        format!("The capability '{cap}' was denied or invalidated because the daemon binary hash changed. Please re-grant or toggle process execution permission in OpenIDEA Settings > Plugins."),
                    ),
                    _ => (
                        "Discord RPC Daemon Missing",
                        "The native daemon 'openidea-discord-rpc-daemon' was not found on your PATH. Install it to enable Discord Rich Presence.".to_string(),
                    ),
                };

                if host::notify_with_actions(
                    NotificationLevel::Warning,
                    title,
                    &body,
                    &[action],
                ).is_err() {
                    host::notify(NotificationLevel::Warning, title, &body);
                }
            }
            DaemonStatus::Outdated => {
                let action = NotificationAction {
                    label: "Update Guide".to_string(),
                    command: "discord-rpc.install-help".to_string(),
                    argument: None,
                };
                let body = format!(
                    "Companion daemon is outdated (v{} installed, v{} required). Update the binary on your PATH and re-approve plugin permissions if prompted.",
                    if self.installed_version.is_empty() { "unknown" } else { &self.installed_version },
                    MIN_DAEMON_VERSION
                );
                if host::notify_with_actions(
                    NotificationLevel::Warning,
                    "Discord RPC Daemon Outdated",
                    &body,
                    &[action],
                ).is_err() {
                    host::notify(NotificationLevel::Warning, "Discord RPC Daemon Outdated", &body);
                }
            }
            DaemonStatus::Connected => {
                // Clear any lingering error notification state
            }
            DaemonStatus::Unknown => {}
        }
    }

    fn verify_or_refresh_connection(&mut self) -> bool {
        match self.check_daemon_version() {
            Ok(v) => {
                // If stdout was empty or status non-zero, treat as outdated (e.g. legacy v0.1.0 daemon without --version)
                if v.is_empty() || is_version_outdated(&v, MIN_DAEMON_VERSION) {
                    // The daemon responded, so it is reachable; only the version is wrong.
                    self.set_daemon_status(DaemonStatus::Outdated, Some(v), None);
                    true
                } else {
                    self.set_daemon_status(DaemonStatus::Connected, Some(v), None);
                    true
                }
            }
            Err(e) => {
                self.set_daemon_status(DaemonStatus::Missing, None, Some(&e));
                false
            }
        }
    }

    fn send_to_daemon(&mut self, args: &[String]) -> bool {
        let result = host::run_process("openidea-discord-rpc-daemon", args, None);
        match result {
            Ok(output) => {
                if output.status != Some(0) {
                    return false;
                }
                if self.daemon_status == DaemonStatus::Missing
                    || self.daemon_status == DaemonStatus::Outdated
                    || self.daemon_status == DaemonStatus::Unknown
                {
                    self.verify_or_refresh_connection();
                }
                true
            }
            Err(e) => {
                self.set_daemon_status(DaemonStatus::Missing, None, Some(&e));
                false
            }
        }
    }
}

impl Plugin for DiscordRpcBridge {
    fn activate(&mut self, _: &Context) -> Result<(), Error> {
        let _ = host::subscribe(EventKind::EditorFocused);
        let _ = host::subscribe(EventKind::FileOpened);
        let _ = host::subscribe(EventKind::VcsChanged);
        let _ = host::subscribe(EventKind::SettingsChanged);

        self.paused = false;
        self.daemon_status = DaemonStatus::Unknown;
        self.installed_version.clear();
        self.notified_state = None;
        self.last_sent = None;
        self.update_ui();

        // Verify daemon availability and version
        let is_ok = self.verify_or_refresh_connection();

        // Start or ensure daemon is running upon plugin activation and provide initial settings
        let settings = current_settings_json(self.paused);
        let mut args = vec![
            "--topic".to_string(),
            "plugin-activated".to_string(),
            "--payload".to_string(),
            settings,
        ];

        let bridge_settings = load_bridge_settings();
        if !bridge_settings.privacy_mode {
            if let Ok(projects) = host::projects() {
                if let Some(first_project) = projects.first() {
                    if let Ok(root) = first_project.root() {
                        args.push("--project".to_string());
                        args.push(root);
                    }
                }
            }
        }

        if is_ok {
            let _ = host::run_process("openidea-discord-rpc-daemon", &args, None);
        }
        Ok(())
    }

    fn deactivate(&mut self) {
        // Stop daemon and clear presence when plugin is deactivated or IDE quits
        let _ = host::run_process(
            "openidea-discord-rpc-daemon",
            &["--shutdown".to_string()],
            None,
        );
    }

    fn run_command(
        &mut self,
        id: &str,
        _: &Invocation,
        _: Option<&str>,
    ) -> Result<Option<String>, Error> {
        match id {
            "discord-rpc.install-help" => {
                let _ = host::open_url("https://github.com/MathCunha16/OpenIDEA-Discord-RCP/releases/latest");
                let msg = if self.daemon_status == DaemonStatus::Outdated {
                    format!(
                        "Daemon update guide opened in browser. Download v{} for your OS and replace the binary in your PATH.",
                        MIN_DAEMON_VERSION
                    )
                } else {
                    "Download the 'openidea-discord-rpc-daemon' binary for your OS, place it in your system PATH, and run 'Discord RPC: Reconnect'.".to_string()
                };
                host::notify(
                    NotificationLevel::Info,
                    "Discord RPC Setup / Update Guide",
                    &msg,
                );
                Ok(Some("Opened Discord RPC setup guide in browser.".to_string()))
            }
            "discord-rpc.reconnect" => {
                let is_ok = self.verify_or_refresh_connection();
                if is_ok {
                    let _ = host::run_process(
                        "openidea-discord-rpc-daemon",
                        &["--reconnect".to_string()],
                        None,
                    );
                    let settings = current_settings_json(self.paused);
                    self.send_to_daemon(&[
                        "--topic".to_string(),
                        "settings-changed".to_string(),
                        "--payload".to_string(),
                        settings,
                    ]);
                    let info_msg = if self.daemon_status == DaemonStatus::Outdated {
                        format!("Reconnected to daemon, but it is outdated (v{} installed, v{} required).", self.installed_version, MIN_DAEMON_VERSION)
                    } else {
                        format!("Connected to Discord RPC daemon v{} successfully.", self.installed_version)
                    };
                    host::notify(NotificationLevel::Info, "Discord RPC", &info_msg);
                    Ok(Some(info_msg))
                } else {
                    let msg = "Daemon still unavailable: it is missing from PATH or its permission was denied. Run 'Discord RPC: Setup Guide' for instructions.";
                    host::notify(NotificationLevel::Warning, "Discord RPC", msg);
                    Ok(Some(msg.to_string()))
                }
            }
            "discord-rpc.toggle" => {
                self.paused = !self.paused;
                self.update_ui();

                let payload = current_settings_json(self.paused);
                self.send_to_daemon(&[
                    "--topic".to_string(),
                    "settings-changed".to_string(),
                    "--payload".to_string(),
                    payload,
                ]);

                let msg = if self.paused {
                    "Discord Rich Presence paused."
                } else {
                    "Discord Rich Presence resumed."
                };
                Ok(Some(msg.to_string()))
            }
            _ => Err(Error::NotFound(format!("command {id}"))),
        }
    }

    fn handle_events(&mut self, events: &[Event], inv: &Invocation) {
        let bridge_settings = load_bridge_settings();
        let project_root = inv.project().and_then(|p| p.root().ok()).unwrap_or_default();
        let editor_file = inv.editor().map(|e| e.path()).unwrap_or_default();
        let clean_editor_file = if is_ignored_file(&editor_file) {
            String::new()
        } else {
            editor_file
        };

        for event in events {
            let (topic, mut payload, event_file) = match event {
                Event::EditorFocused(path) => {
                    let p = path.as_deref().unwrap_or("");
                    if is_ignored_file(p) {
                        continue;
                    }
                    ("editor-focused", p.to_string(), if !p.is_empty() { Some(p.to_string()) } else { None })
                }
                Event::FileOpened(path) => {
                    if is_ignored_file(path) {
                        continue;
                    }
                    ("file-opened", path.clone(), Some(path.clone()))
                }
                Event::VcsChanged(path) => (
                    "vcs-changed",
                    path.clone(),
                    if !clean_editor_file.is_empty() { Some(clean_editor_file.clone()) } else { None },
                ),
                Event::Scheduled => (
                    "scheduled",
                    current_settings_json(self.paused),
                    if !clean_editor_file.is_empty() { Some(clean_editor_file.clone()) } else { None },
                ),
                Event::Other { kind, data: _ } => {
                    if kind == "settings-changed" {
                        ("settings-changed", current_settings_json(self.paused), None)
                    } else {
                        // Skip unrecognized events without spawning processes
                        continue;
                    }
                }
                _ => continue,
            };

            if topic == "settings-changed" {
                self.update_ui();
                payload = current_settings_json(self.paused);
            }

            if payload.is_empty() && topic != "settings-changed" && topic != "scheduled" {
                if let Some(ref f) = event_file {
                    payload = f.clone();
                }
            }

            // Enforce Privacy Mode directly at WASM boundary to avoid leaking paths in CLI arguments
            let (file_arg, project_arg, effective_payload) = if bridge_settings.privacy_mode {
                (None, None, if topic == "settings-changed" || topic == "scheduled" { payload } else { "Working on a project".to_string() })
            } else {
                (
                    if let Some(ref f) = event_file {
                        if !f.is_empty() && !is_ignored_file(f) { Some(f.clone()) } else { None }
                    } else { None },
                    if !project_root.is_empty() { Some(project_root.clone()) } else { None },
                    payload,
                )
            };

            // Deduplicate: avoid spawning new forwarder process if presence arguments haven't changed
            let dedup_key = (
                topic.to_string(),
                effective_payload.clone(),
                file_arg.clone().unwrap_or_default(),
            );
            if self.last_sent.as_ref() == Some(&dedup_key) && topic != "scheduled" {
                continue;
            }

            let mut args = vec![
                "--topic".to_string(),
                topic.to_string(),
                "--payload".to_string(),
                effective_payload,
            ];

            if let Some(f) = file_arg {
                args.push("--file".to_string());
                args.push(f);
            }

            if let Some(p) = project_arg {
                args.push("--project".to_string());
                args.push(p);
            }

            if self.send_to_daemon(&args) {
                self.last_sent = Some(dedup_key);
            }
        }
    }
}

register_plugin!(DiscordRpcBridge);

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_is_ignored_file() {
        assert!(is_ignored_file("/path/to/.openidea/config"));
        assert!(is_ignored_file("/path/to/.openidea/settings.json"));
        assert!(is_ignored_file("/path/to/settings.json"));
        assert!(is_ignored_file("/path/to/settings.toml"));
        assert!(is_ignored_file("{temp_virtual_doc}"));

        // Must NOT ignore files inside directories that just happen to contain the word .openidea
        assert!(!is_ignored_file("/home/user/.openidea-plugins/my-plugin/src/main.rs"));
        assert!(!is_ignored_file("/path/to/main.rs"));
    }

    #[test]
    fn test_bridge_defaults() {
        let bridge = DiscordRpcBridge::default();
        assert!(!bridge.paused);
        assert_eq!(bridge.daemon_status, DaemonStatus::Unknown);
        assert!(bridge.installed_version.is_empty());
        assert_eq!(bridge.notified_state, None);
        assert!(bridge.last_sent.is_none());
    }

    #[test]
    fn test_parse_version() {
        assert_eq!(parse_version("0.1.0"), Some((0, 1, 0)));
        assert_eq!(parse_version("v0.1.1"), Some((0, 1, 1)));
        assert_eq!(parse_version("1.2.3\n"), Some((1, 2, 3)));
        assert_eq!(parse_version("openidea-discord-rpc-daemon 0.1.1"), Some((0, 1, 1)));
        assert_eq!(parse_version("daemon v0.1.1-beta.1"), Some((0, 1, 1)));
        assert_eq!(parse_version("invalid text"), None);
    }

    #[test]
    fn test_is_version_outdated() {
        assert!(is_version_outdated("0.1.0", "0.1.1"));
        assert!(is_version_outdated("0.1.0", "0.2.0"));
        assert!(is_version_outdated("0.9.9", "1.0.0"));
        assert!(!is_version_outdated("0.1.1", "0.1.1"));
        assert!(!is_version_outdated("0.1.2", "0.1.1"));
        assert!(!is_version_outdated("1.0.0", "0.1.1"));
    }
}


