package app

import (
	"encoding/binary"
	"fmt"
	"strings"

	"wit_component/openidea_plugin_ipc"
	"wit_component/openidea_plugin_types"
)

// SocketCandidates lists the socket paths across platforms to probe for Discord desktop.
// Note: OpenIDEA expands $XDG_RUNTIME_DIR and $TMPDIR in ipc: grants.
var SocketCandidates = []string{
	`\\.\pipe\discord-ipc-%d`,
	"$XDG_RUNTIME_DIR/discord-ipc-%d",
	"$XDG_RUNTIME_DIR/app/com.discordapp.Discord/discord-ipc-%d",
	"$XDG_RUNTIME_DIR/app/dev.vencord.Vesktop/discord-ipc-%d",
	"$XDG_RUNTIME_DIR/.flatpak/com.discordapp.Discord/xdg-run/discord-ipc-%d",
	"$XDG_RUNTIME_DIR/.flatpak/dev.vencord.Vesktop/xdg-run/discord-ipc-%d",
	"$XDG_RUNTIME_DIR/app/com.discordapp.DiscordCanary/discord-ipc-%d",
	"$XDG_RUNTIME_DIR/app/com.discordapp.DiscordPTB/discord-ipc-%d",
	"$XDG_RUNTIME_DIR/snap.discord/discord-ipc-%d",
	"$TMPDIR/discord-ipc-%d",
	"/tmp/discord-ipc-%d",
}

// IPCClient encapsulates the low-level Discord IPC socket connection over OpenIDEA's host::ipc.
type IPCClient struct {
	conn       *openidea_plugin_ipc.Connection
	lastSocket string
}

// NewIPCClient initializes an unbound IPC client.
func NewIPCClient() *IPCClient {
	return &IPCClient{}
}

// IsConnected returns true if an active handle is held.
func (c *IPCClient) IsConnected() bool {
	return c != nil && c.conn != nil
}

// Close gracefully terminates the connection if open.
func (c *IPCClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

// Connect probes known Discord socket locations or tests a preferred lastSocket.
func (c *IPCClient) Connect(preferredSocket string) error {
	c.Close()

	if preferredSocket != "" {
		if conn, err := tryDial(preferredSocket); err == nil {
			c.conn = conn
			c.lastSocket = preferredSocket
			return nil
		}
	}

	var lastErr string
	for _, pattern := range SocketCandidates {
		for i := 0; i < 2; i++ {
			candidate := fmt.Sprintf(pattern, i)
			conn, err := tryDial(candidate)
			if err == nil {
				c.conn = conn
				c.lastSocket = candidate
				return nil
			}
			lastErr = err.Error()
		}
	}

	return fmt.Errorf("no Discord IPC socket reachable (last error: %s)", lastErr)
}

func tryDial(path string) (*openidea_plugin_ipc.Connection, error) {
	r := openidea_plugin_ipc.Connect(path)
	if r.IsOk() {
		return r.Ok(), nil
	}
	return nil, fmt.Errorf("%+v", r.Err())
}

// LastSocket returns the last successfully connected socket path.
func (c *IPCClient) LastSocket() string {
	return c.lastSocket
}

// SendRawFrame transmits a raw byte buffer (header + payload) over the IPC resource.
func (c *IPCClient) SendRawFrame(frame []byte) error {
	if !c.IsConnected() {
		return fmt.Errorf("discord ipc: not connected")
	}
	res := c.conn.Write(frame)
	if res.IsErr() {
		c.Close()
		return fmt.Errorf("discord ipc write failed: %s", formatError(res.Err()))
	}
	return nil
}

// ReadFrame decodes an 8-byte little-endian header and reads the exact payload bytes.
func (c *IPCClient) ReadFrame(timeoutMs uint32) (uint32, []byte, error) {
	if !c.IsConnected() {
		return 0, nil, fmt.Errorf("discord ipc: not connected")
	}

	header, err := c.readFull(8, timeoutMs)
	if err != nil {
		c.Close()
		return 0, nil, fmt.Errorf("discord ipc read header: %w", err)
	}

	opcode := binary.LittleEndian.Uint32(header[0:4])
	length := binary.LittleEndian.Uint32(header[4:8])

	if length > 1024*1024 {
		c.Close()
		return 0, nil, fmt.Errorf("discord ipc payload too large: %d bytes", length)
	}

	payload, err := c.readFull(int(length), timeoutMs)
	if err != nil {
		c.Close()
		return 0, nil, fmt.Errorf("discord ipc read payload: %w", err)
	}

	return opcode, payload, nil
}

// readFull guarantees reading exactly n bytes, looping over chunk reads.
// Wasmtime's host IPC returns empty slice on timeout or partial read.
func (c *IPCClient) readFull(n int, timeoutMs uint32) ([]byte, error) {
	buf := make([]byte, 0, n)
	retries := 0
	maxRetries := 5

	for len(buf) < n {
		want := n - len(buf)
		if want > 1<<20 {
			want = 1 << 20
		}

		res := c.conn.Read(uint32(want), timeoutMs)
		if res.IsErr() {
			return nil, fmt.Errorf("read error: %s", formatError(res.Err()))
		}

		chunk := res.Ok()
		if len(chunk) == 0 {
			retries++
			if retries >= maxRetries {
				return nil, fmt.Errorf("read timeout waiting for %d bytes (received %d)", n, len(buf))
			}
			continue
		}

		retries = 0 // Reset retry count upon successfully receiving data chunk
		buf = append(buf, chunk...)
	}

	return buf, nil
}

func formatError(err openidea_plugin_types.Error) string {
	switch err.Tag() {
	case 0:
		return fmt.Sprintf("denied(%s)", err.Denied())
	case 1:
		return "stale"
	case 2:
		return fmt.Sprintf("not_found(%s)", err.NotFound())
	case 3:
		return fmt.Sprintf("invalid(%s)", err.Invalid())
	case 4:
		return "cancelled"
	case 5:
		return fmt.Sprintf("failed(%s)", err.Failed())
	default:
		return "unknown"
	}
}

// ConvertError formats an error into OpenIDEA's exported types.Error
func ConvertError(err error) openidea_plugin_types.Error {
	if err == nil {
		return openidea_plugin_types.MakeErrorFailed("unknown")
	}
	msg := err.Error()
	if strings.Contains(msg, "denied") {
		return openidea_plugin_types.MakeErrorDenied(msg)
	}
	return openidea_plugin_types.MakeErrorFailed(msg)
}
