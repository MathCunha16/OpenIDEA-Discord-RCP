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

    fn handle_events(&mut self, events: &[Event], _: &Invocation) {
        for event in events {
            let (topic, payload) = match event {
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

            let _ = host::run_process(
                "openidea-discord-rpc-daemon",
                &[
                    "--topic".to_string(),
                    topic.to_string(),
                    "--payload".to_string(),
                    payload,
                ],
                None,
            );
        }
    }
}

register_plugin!(DiscordRpcBridge);
