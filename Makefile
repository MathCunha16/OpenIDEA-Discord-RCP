.PHONY: all build test clean wasm package sync-version bump

VERSION := 0.3.0
PLUGIN_NAME=cunha.discord-rpc-$(VERSION).oiplugin

COMPONENTIZE_GO ?= $(shell command -v componentize-go 2>/dev/null || echo $(shell go env GOPATH)/bin/componentize-go)
GO_BIN_PATH ?= $(shell go env GOPATH)/bin:$(PATH)

all: test build

## Bump the project version everywhere: make bump VERSION=x.y.z
bump:
	@sed -i.bak -E 's/^(VERSION := )[^ ]+/\1'$(VERSION)'/' Makefile
	@rm -f Makefile.bak
	@$(MAKE) sync-version

## Sync Makefile VERSION into plugin.toml
sync-version:
	@sed -i.bak -E 's/^version = "[^"]*"/version = "'$(VERSION)'"/' plugin.toml
	@sed -i.bak -E 's/^(change-notes = ")[0-9]+\.[0-9]+\.[0-9]+( —)/\1'$(VERSION)'\2/' plugin.toml
	@rm -f plugin.toml.bak
	@echo "Synced version $(VERSION) into plugin.toml"

## Build the WASM plugin
build: wasm

## Run all automated unit and race tests
test:
	go test -count=1 -v -race ./...
	cd wasm/gen && go test -count=1 -v -race ./app/...

## Compile the WASM Component Model plugin for OpenIDEA using componentize-go
wasm: sync-version
	@mkdir -p dist
	cd wasm/gen && PATH="$(GO_BIN_PATH):$$PATH" $(COMPONENTIZE_GO) -d ../../wit -w plugin build -o ../../dist/plugin.wasm

## Package the .oiplugin archive for OpenIDEA installation
package: wasm
	@mkdir -p dist
	@rm -f $(PLUGIN_NAME)
	zip -r $(PLUGIN_NAME) plugin.toml schemas/ icon.svg dist/ assets/ README.md LICENSE
	@echo "Packaged plugin bundle: $(PLUGIN_NAME)"

## Clean compiled binaries and temporary outputs
clean:
	rm -rf bin/ dist/ *.oiplugin
