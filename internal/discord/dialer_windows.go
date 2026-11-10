//go:build windows

package discord

// DefaultDialer returns the platform-specific SocketDialer (WindowsDialer on Windows).
func DefaultDialer() SocketDialer {
	return NewWindowsDialer()
}
