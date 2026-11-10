#[allow(warnings)]
mod bindings;

use bindings::exports::event_handler::Guest as EventHandlerGuest;
use bindings::exports::lifecycle::Guest as LifecycleGuest;
use bindings::openidea::plugin::events;
use bindings::openidea::plugin::process;

struct Component;

impl LifecycleGuest for Component {
    fn activate() {
        let _ = events::subscribe("editor-focused");
        let _ = events::subscribe("file-opened");
        let _ = events::subscribe("files-changed");
        let _ = events::subscribe("vcs-changed");
    }

    fn deactivate() {
        let _ = process::run("openidea-discord-rpc-daemon", &["--shutdown".to_string()], None, None);
    }
}

impl EventHandlerGuest for Component {
    fn handle(topic: String, payload: String) {
        let _ = process::run(
            "openidea-discord-rpc-daemon",
            &["--topic".to_string(), topic, "--payload".to_string(), payload],
            None,
            None,
        );
    }
}

bindings::export!(Component with_types_in bindings);
