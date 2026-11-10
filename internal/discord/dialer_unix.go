//go:build !windows

package discord

// DefaultDialer returns the platform-specific SocketDialer (UnixDialer on non-Windows).
func DefaultDialer() SocketDialer {
	return NewUnixDialer()
}
