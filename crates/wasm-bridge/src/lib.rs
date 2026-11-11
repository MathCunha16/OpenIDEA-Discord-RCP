use openidea_plugin::{
    Context, Error, Event, EventKind, Invocation, Plugin, StatusItem, host, register_plugin,
};

#[derive(Default)]
struct DiscordRpcBridge {
    paused: bool,
}

const STATUS_BAR_ID: &str = "discord-rpc-status";

fn update_status_bar(enabled: bool, paused: bool) {
    let (text, tooltip) = if !enabled {
        ("○ Discord (Off)", "Discord Rich Presence: Disabled in settings")
    } else if paused {
        ("○ Discord (Paused)", "Discord Rich Presence: Paused (Click to resume)")
    } else {
        ("● Discord", "Discord Rich Presence: Active (Click to pause)")
    };

    host::set_status_item(
        STATUS_BAR_ID,
        &StatusItem {
            text: text.to_string(),
            tooltip: Some(tooltip.to_string()),
            command: Some("discord-rpc.toggle".to_string()),
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

fn current_settings_json() -> String {
    let settings = BridgeSettings {
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
        file_time_mode: host::setting_string("fileTimeMode").unwrap_or_else(|| "Project Time".to_string()),
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
    };

    serde_json::to_string(&settings).unwrap_or_default()
}

fn is_ignored_file(path: &str) -> bool {
    let lower = path.to_lowercase();
    lower.ends_with("settings.json")
        || lower.ends_with("settings.toml")
        || lower.contains(".openidea")
        || lower.starts_with('{')
}

impl Plugin for DiscordRpcBridge {
    fn activate(&mut self, _: &Context) -> Result<(), Error> {
        let _ = host::subscribe(EventKind::EditorFocused);
        let _ = host::subscribe(EventKind::FileOpened);
        let _ = host::subscribe(EventKind::FilesChanged);
        let _ = host::subscribe(EventKind::VcsChanged);
        let _ = host::subscribe(EventKind::SettingsChanged);

        let enabled = host::setting_bool("enabled").unwrap_or(true);
        self.paused = false;
        update_status_bar(enabled, self.paused);

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

        let _ = host::run_process("openidea-discord-rpc-daemon", &args, None);
        Ok(())
    }

    fn deactivate(&mut self) {
        host::remove_status_item(STATUS_BAR_ID);
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
            "discord-rpc.reconnect" => {
                let _ = host::run_process(
                    "openidea-discord-rpc-daemon",
                    &["--reconnect".to_string()],
                    None,
                );
                let enabled = host::setting_bool("enabled").unwrap_or(true);
                update_status_bar(enabled, self.paused);
                Ok(Some("Discord RPC reconnect triggered".to_string()))
            }
            "discord-rpc.toggle" => {
                self.paused = !self.paused;
                let enabled = host::setting_bool("enabled").unwrap_or(true);
                update_status_bar(enabled, self.paused);

                let mut settings: BridgeSettings = serde_json::from_str(&current_settings_json()).unwrap();
                if self.paused {
                    settings.enabled = false;
                }
                let payload = serde_json::to_string(&settings).unwrap_or_default();

                let _ = host::run_process(
                    "openidea-discord-rpc-daemon",
                    &[
                        "--topic".to_string(),
                        "settings-changed".to_string(),
                        "--payload".to_string(),
                        payload,
                    ],
                    None,
                );

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
                Event::FilesChanged(_) => (
                    "files-changed",
                    "".to_string(),
                    if !clean_editor_file.is_empty() { Some(clean_editor_file.clone()) } else { None },
                ),
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
                let enabled = host::setting_bool("enabled").unwrap_or(true);
                update_status_bar(enabled, self.paused);
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

            let _ = host::run_process("openidea-discord-rpc-daemon", &args, None);
        }
    }
}

register_plugin!(DiscordRpcBridge);

