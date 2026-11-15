package export_openidea_plugin_event_handler

import (
	"wit_component/app"
	"wit_component/openidea_plugin_events"
	"wit_component/openidea_plugin_invocation"
)

func Handle(events []openidea_plugin_events.Event, ctx *openidea_plugin_invocation.Invocation) {
	if ctx != nil {
		defer ctx.Drop()
	}
	app.GetInstance().Handle(events, ctx)
}
