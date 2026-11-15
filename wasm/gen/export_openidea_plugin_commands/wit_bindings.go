package export_openidea_plugin_commands

import (
	"wit_component/app"
	"wit_component/openidea_plugin_invocation"
	"wit_component/openidea_plugin_types"

	witTypes "go.bytecodealliance.org/pkg/wit/types"
)

func Run(id string, ctx *openidea_plugin_invocation.Invocation, arg witTypes.Option[string]) witTypes.Result[witTypes.Option[string], openidea_plugin_types.Error] {
	if ctx != nil {
		defer ctx.Drop()
	}
	return app.GetInstance().Run(id, ctx, arg)
}
