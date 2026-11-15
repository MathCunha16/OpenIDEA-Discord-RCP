package export_openidea_plugin_lifecycle

import (
	"wit_component/app"
	"wit_component/openidea_plugin_types"

	witTypes "go.bytecodealliance.org/pkg/wit/types"
)

func Activate(ctx openidea_plugin_types.Context) witTypes.Result[witTypes.Unit, openidea_plugin_types.Error] {
	return app.GetInstance().Activate(ctx)
}

func Deactivate() {
	app.GetInstance().Deactivate()
}
