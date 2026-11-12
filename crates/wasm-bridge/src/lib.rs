use openidea_plugin::{
    Context, Error, Event, EventKind, Invocation, NotificationAction, NotificationLevel, Plugin,
    StatusItem, host, register_plugin,
};

#[derive(Default)]
struct DiscordRpcBridge {
    paused: bool,
    daemon_missing: bool,
    daemon_notified: bool,
}

const STATUS_BAR_ID: &str = "discord-rpc-status";

fn update_status_bar(enabled: bool, paused: bool, daemon_missing: bool) {
    let (text, tooltip, command) = if daemon_missing {
        (
            "⚠ Discord (Daemon Missing)",
            "Discord RPC daemon not found on PATH. Click for setup instructions.",
            "discord-rpc.install-help",
        )
    } else if !enabled {
        (
            "○ Discord (Off)",
            "Discord Rich Presence: Disabled in settings",
            "discord-rpc.toggle",
        )
    } else if paused {
        (
            "○ Discord (Paused)",
            "Discord Rich Presence: Paused (Click to resume)",
            "discord-rpc.toggle",
        )
    } else {
        (
            "● Discord",
            "Discord Rich Presence: Active (Click to pause)",
            "discord-rpc.toggle",
        )
    };

    host::set_status_item(
        STATUS_BAR_ID,
        &StatusItem {
            text: text.to_string(),
            tooltip: Some(tooltip.to_string()),
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

fn current_settings_json() -> String {
    let settings = load_bridge_settings();
    serde_json::to_string(&settings).unwrap_or_default()
}

fn is_ignored_file(path: &str) -> bool {
    let lower = path.to_lowercase();
    lower.ends_with("settings.json")
        || lower.ends_with("settings.toml")
        || lower.contains(".openidea")
        || lower.starts_with('{')
}

impl DiscordRpcBridge {
    fn update_ui(&self) {
        let enabled = host::setting_bool("enabled").unwrap_or(true);
        update_status_bar(enabled, self.paused, self.daemon_missing);
    }

    fn send_to_daemon(&mut self, args: &[String]) {
        let result = host::run_process("openidea-discord-rpc-daemon", args, None);
        match result {
            Ok(_) => {
                if self.daemon_missing {
                    self.daemon_missing = false;
                    self.daemon_notified = false;
                    self.update_ui();
                }
            }
            Err(_) => {
                self.daemon_missing = true;
                self.update_ui();
                if !self.daemon_notified {
                    self.daemon_notified = true;
                    let action = NotificationAction {
                        label: "Setup Guide".to_string(),
                        command: "discord-rpc.install-help".to_string(),
                        argument: None,
                    };
                    if host::notify_with_actions(
                        NotificationLevel::Warning,
                        "Discord RPC Daemon Missing",
                        "The native daemon 'openidea-discord-rpc-daemon' was not found on your PATH. Install it to enable Discord Rich Presence.",
                        &[action],
                    ).is_err() {
                        host::notify(
                            NotificationLevel::Warning,
                            "Discord RPC Daemon Missing",
                            "The native daemon 'openidea-discord-rpc-daemon' was not found on your PATH. Install it to enable Discord Rich Presence.",
                        );
                    }
                }
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
        self.daemon_missing = false;
        self.daemon_notified = false;
        self.update_ui();

        // Start or ensure daemon is running upon plugin activation and provide initial settings
        let settings = current_settings_json();
        let mut args = vec![
            "--topic".to_string(),
            "plugin-activated".to_string(),
            "--payload".to_string(),
            settings,
        ];

        if let Ok(projects) = host::projects() {
            if let Some(first_project) = projects.first() {
                if let Ok(root) = first_project.root() {
                    args.push("--project".to_string());
                    args.push(root);
                }
            }
        }

        self.send_to_daemon(&args);
        Ok(())
    }

    fn deactivate(&mut self) {
        // Idle reclaim: do not kill the sidecar daemon or remove the status bar.
        // The sidecar daemon manages its own Discord session and AFK timers.
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
                host::notify(
                    NotificationLevel::Info,
                    "Discord RPC Installation Guide",
                    "Download the 'openidea-discord-rpc-daemon' binary for your OS, place it in your system PATH, and run 'Discord RPC: Reconnect'.",
                );
                Ok(Some("Opened Discord RPC installation guide in browser.".to_string()))
            }
            "discord-rpc.reconnect" => {
                let result = host::run_process(
                    "openidea-discord-rpc-daemon",
                    &["--reconnect".to_string()],
                    None,
                );
                match result {
                    Ok(_) => {
                        self.daemon_missing = false;
                        self.daemon_notified = false;
                        self.update_ui();
                        host::notify(
                            NotificationLevel::Info,
                            "Discord RPC",
                            "Connected to Discord RPC daemon successfully.",
                        );
                        Ok(Some("Discord RPC reconnected successfully.".to_string()))
                    }
                    Err(e) => {
                        self.daemon_missing = true;
                        self.daemon_notified = true;
                        self.update_ui();
                        host::notify(
                            NotificationLevel::Warning,
                            "Discord RPC Daemon Missing",
                            "Daemon 'openidea-discord-rpc-daemon' was not found on PATH. Please follow the setup guide.",
                        );
                        Ok(Some(format!("Daemon reconnection failed: {e:?}")))
                    }
                }
            }
            "discord-rpc.toggle" => {
                self.paused = !self.paused;
                self.update_ui();

                let mut settings = load_bridge_settings();
                if self.paused {
                    settings.enabled = false;
                }
                let payload = serde_json::to_string(&settings).unwrap_or_default();

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
                Event::FileSaved(path) => {
                    if is_ignored_file(path) {
                        continue;
                    }
                    ("file-saved", path.clone(), Some(path.clone()))
                }
                Event::VcsChanged(path) => (
                    "vcs-changed",
                    path.clone(),
                    if !clean_editor_file.is_empty() { Some(clean_editor_file.clone()) } else { None },
                ),
                Event::Scheduled => (
                    "scheduled",
                    "".to_string(),
                    if !clean_editor_file.is_empty() { Some(clean_editor_file.clone()) } else { None },
                ),
                Event::ActionPerformed(_) => (
                    "action-performed",
                    "".to_string(),
                    if !clean_editor_file.is_empty() { Some(clean_editor_file.clone()) } else { None },
                ),
                Event::Other { kind, data } => {
                    if kind == "settings-changed" {
                        ("settings-changed", current_settings_json(), None)
                    } else {
                        (
                            kind.as_str(),
                            data.clone(),
                            if !clean_editor_file.is_empty() { Some(clean_editor_file.clone()) } else { None },
                        )
                    }
                }
                _ => ("unknown", "".to_string(), None),
            };

            if topic == "settings-changed" {
                self.update_ui();
                payload = current_settings_json();
            }

            if payload.is_empty() && topic != "settings-changed" {
                if let Some(ref f) = event_file {
                    payload = f.clone();
                }
            }

            let mut args = vec![
                "--topic".to_string(),
                topic.to_string(),
                "--payload".to_string(),
                payload,
            ];

            if let Some(ref f) = event_file {
                if !f.is_empty() && !is_ignored_file(f) {
                    args.push("--file".to_string());
                    args.push(f.clone());
                }
            }

            if !project_root.is_empty() {
                args.push("--project".to_string());
                args.push(project_root.clone());
            }

            self.send_to_daemon(&args);
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
        assert!(is_ignored_file("/path/to/settings.json"));
        assert!(is_ignored_file("/path/to/settings.toml"));
        assert!(!is_ignored_file("/path/to/main.rs"));
    }

    #[test]
    fn test_bridge_defaults() {
        let bridge = DiscordRpcBridge::default();
        assert!(!bridge.paused);
        assert!(!bridge.daemon_missing);
        assert!(!bridge.daemon_notified);
    }
}

