//go:build !wasip2

package app

import (
	"wit_component/openidea_plugin_types"

	witTypes "go.bytecodealliance.org/pkg/wit/types"
)

func hostStorageGet(_ string) witTypes.Result[witTypes.Option[string], openidea_plugin_types.Error] {
	return witTypes.Ok[witTypes.Option[string], openidea_plugin_types.Error](witTypes.None[string]())
}

func hostStorageSet(_, _ string) witTypes.Result[witTypes.Unit, openidea_plugin_types.Error] {
	return witTypes.Ok[witTypes.Unit, openidea_plugin_types.Error](witTypes.Unit{})
}

func hostStorageDelete(_ string) witTypes.Result[witTypes.Unit, openidea_plugin_types.Error] {
	return witTypes.Ok[witTypes.Unit, openidea_plugin_types.Error](witTypes.Unit{})
}
