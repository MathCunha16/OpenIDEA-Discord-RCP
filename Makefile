.PHONY: all build test clean daemon package

BINARY_NAME=openidea-discord-rpc-daemon
PLUGIN_NAME=cunha.discord-rpc-0.1.0.oiplugin

all: test build

## Run all automated unit and race tests
test:
	go test -count=1 -v -race ./...

## Compile the native sidecar daemon for current platform
daemon:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/$(BINARY_NAME) ./cmd/daemon

## Cross-compile native daemon for all supported platforms
cross-compile:
	@mkdir -p bin/dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/dist/$(BINARY_NAME)-linux-amd64 ./cmd/daemon
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/dist/$(BINARY_NAME)-darwin-amd64 ./cmd/daemon
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/dist/$(BINARY_NAME)-darwin-arm64 ./cmd/daemon
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/daemon

## Compile the WASM Component Model bridge for OpenIDEA
wasm:
	cargo build --release --manifest-path crates/wasm-bridge/Cargo.toml --target wasm32-wasip2
	@mkdir -p dist
	cp crates/wasm-bridge/target/wasm32-wasip2/release/openidea_discord_rpc_bridge.wasm dist/plugin.wasm

## Package the .oiplugin archive for OpenIDEA installation
package: daemon wasm
	@mkdir -p dist
	@rm -f $(PLUGIN_NAME)
	zip -r $(PLUGIN_NAME) plugin.toml schemas/ icon.svg dist/ bin/ assets/ README.md
	@echo "Packaged plugin bundle: $(PLUGIN_NAME)"

## Clean compiled binaries and temporary outputs
clean:
	rm -rf bin/ dist/ *.oiplugin
