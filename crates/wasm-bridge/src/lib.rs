use openidea_plugin::{
    Context, Error, Event, EventKind, Invocation, Plugin, host, register_plugin,
};

#[derive(Default)]
struct DiscordRpcBridge;

impl Plugin for DiscordRpcBridge {
    fn activate(&mut self, _: &Context) -> Result<(), Error> {
        let _ = host::subscribe(EventKind::EditorFocused);
        let _ = host::subscribe(EventKind::FileOpened);
        let _ = host::subscribe(EventKind::FilesChanged);
        let _ = host::subscribe(EventKind::VcsChanged);

        // Start or ensure daemon is running upon plugin activation
        let _ = host::run_process(
            "openidea-discord-rpc-daemon",
            &["--topic".to_string(), "plugin-activated".to_string()],
            None,
        );
        Ok(())
    }

    fn deactivate(&mut self) {
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
                Ok(Some("Discord RPC reconnect triggered".to_string()))
            }
            _ => Err(Error::NotFound(format!("command {id}"))),
        }
    }

    fn handle_events(&mut self, events: &[Event], inv: &Invocation) {
        let project_root = inv.project().and_then(|p| p.root().ok()).unwrap_or_default();
        let editor_file = inv.editor().map(|e| e.path()).unwrap_or_default();

        for event in events {
            let (topic, mut payload) = match event {
                Event::EditorFocused(path) => (
                    "editor-focused",
                    path.as_deref().unwrap_or("").to_string(),
                ),
                Event::FileOpened(path) => ("file-opened", path.clone()),
                Event::FileSaved(path) => ("file-saved", path.clone()),
                Event::FilesChanged(_) => ("files-changed", "".to_string()),
                Event::VcsChanged(path) => ("vcs-changed", path.clone()),
                Event::Scheduled => ("scheduled", "".to_string()),
                Event::ActionPerformed(_) => ("action-performed", "".to_string()),
                Event::Other { kind, data } => (kind.as_str(), data.clone()),
                _ => ("unknown", "".to_string()),
            };

            if payload.is_empty() && !editor_file.is_empty() {
                payload = editor_file.clone();
            }

            let mut args = vec![
                "--topic".to_string(),
                topic.to_string(),
                "--payload".to_string(),
                payload,
            ];

            if !editor_file.is_empty() {
                args.push("--file".to_string());
                args.push(editor_file.clone());
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
