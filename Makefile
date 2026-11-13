.PHONY: all build test clean daemon cross-compile checksums wasm package sync-version bump

VERSION := 0.2.1
BINARY_NAME=openidea-discord-rpc-daemon
PLUGIN_NAME=cunha.discord-rpc-$(VERSION).oiplugin
LDFLAGS=-s -w -X main.DaemonVersion=$(VERSION)

all: test build

## Bump the project version everywhere: make bump VERSION=x.y.z
bump:
	@sed -i.bak -E 's/^(VERSION := )[^ ]+/\1'$(VERSION)'/' Makefile
	@rm -f Makefile.bak
	@$(MAKE) sync-version

## Sync Makefile VERSION into plugin.toml, Cargo.toml and Cargo.lock
sync-version:
	@sed -i.bak -E 's/^version = "[^"]*"/version = "'$(VERSION)'"/' plugin.toml
	@sed -i.bak -E 's/^(change-notes = ")[0-9]+\.[0-9]+\.[0-9]+( —)/\1'$(VERSION)'\2/' plugin.toml
	@sed -i.bak -E 's/^version = "[^"]*"/version = "'$(VERSION)'"/' crates/wasm-bridge/Cargo.toml
	@sed -i.bak -E '/^name = "openidea-discord-rpc-bridge"$$/,/^version = / s/^version = "[^"]*"/version = "'$(VERSION)'"/' crates/wasm-bridge/Cargo.lock
	@rm -f plugin.toml.bak crates/wasm-bridge/Cargo.toml.bak crates/wasm-bridge/Cargo.lock.bak
	@echo "Synced version $(VERSION) into plugin.toml, Cargo.toml and Cargo.lock"

## Build all targets for development
build: daemon wasm

## Run all automated unit and race tests
test:
	go test -count=1 -v -race ./...

## Compile the native sidecar daemon for current platform
daemon:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/daemon

## Cross-compile native daemon for all supported platforms
cross-compile:
	@mkdir -p bin/dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/dist/$(BINARY_NAME)-linux-amd64 ./cmd/daemon
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/dist/$(BINARY_NAME)-linux-arm64 ./cmd/daemon
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/dist/$(BINARY_NAME)-darwin-amd64 ./cmd/daemon
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/dist/$(BINARY_NAME)-darwin-arm64 ./cmd/daemon
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS) -H=windowsgui" -o bin/dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/daemon

## Generate SHA-256 checksums for cross-compiled daemon binaries
checksums: cross-compile
	@cd bin/dist && rm -f checksums.txt && sha256sum $(BINARY_NAME)-linux-amd64 $(BINARY_NAME)-linux-arm64 $(BINARY_NAME)-darwin-amd64 $(BINARY_NAME)-darwin-arm64 $(BINARY_NAME)-windows-amd64.exe > checksums.txt
	@echo "Generated bin/dist/checksums.txt:"
	@cat bin/dist/checksums.txt

## Compile the WASM Component Model bridge for OpenIDEA
wasm: sync-version
	cargo build --release --manifest-path crates/wasm-bridge/Cargo.toml --target wasm32-wasip2
	@mkdir -p dist
	cp crates/wasm-bridge/target/wasm32-wasip2/release/openidea_discord_rpc_bridge.wasm dist/plugin.wasm

## Package the .oiplugin archive for OpenIDEA installation
package: wasm
	@mkdir -p dist
	@rm -f $(PLUGIN_NAME)
	zip -r $(PLUGIN_NAME) plugin.toml schemas/ icon.svg dist/ assets/ README.md
	@echo "Packaged plugin bundle: $(PLUGIN_NAME)"

## Clean compiled binaries and temporary outputs
clean:
	rm -rf bin/ dist/ *.oiplugin
