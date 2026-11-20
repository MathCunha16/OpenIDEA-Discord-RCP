//go:build wasip2

package app

import (
	"wit_component/openidea_plugin_storage"
	"wit_component/openidea_plugin_types"

	witTypes "go.bytecodealliance.org/pkg/wit/types"
)

func hostStorageGet(key string) witTypes.Result[witTypes.Option[string], openidea_plugin_types.Error] {
	return openidea_plugin_storage.Get(key)
}

func hostStorageSet(key, val string) witTypes.Result[witTypes.Unit, openidea_plugin_types.Error] {
	return openidea_plugin_storage.Set(key, val)
}

func hostStorageDelete(key string) witTypes.Result[witTypes.Unit, openidea_plugin_types.Error] {
	return openidea_plugin_storage.Delete(key)
}
