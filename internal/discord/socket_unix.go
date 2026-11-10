//go:build !windows

package discord

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ErrSocketNotFound indicates that no responsive Discord IPC socket was discovered.
var ErrSocketNotFound = errors.New("discord: no active discord socket found")

// UnixDialer searches and connects to Unix domain sockets used by Discord, Vesktop, Flatpak, and macOS.
type UnixDialer struct {
	// CustomPatterns can override default search patterns (primarily useful in tests).
	CustomPatterns []string
	// Timeout per socket dial attempt.
	DialTimeout time.Duration
}

// NewUnixDialer creates a UnixDialer with standard timeout defaults.
func NewUnixDialer() *UnixDialer {
	return &UnixDialer{
		DialTimeout: 500 * time.Millisecond,
	}
}

// getSocketPatterns builds prioritized socket path templates for Linux and macOS.
func (d *UnixDialer) getSocketPatterns() []string {
	if len(d.CustomPatterns) > 0 {
		return d.CustomPatterns
	}

	var patterns []string
	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
	homeDir, _ := os.UserHomeDir()
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}

	if xdgRuntimeDir != "" {
		// 1. Standard XDG runtime socket (Native Discord, Deb, Snap with portal)
		patterns = append(patterns, filepath.Join(xdgRuntimeDir, "discord-ipc-%d"))

		// 2. Vesktop Flatpak paths
		patterns = append(patterns,
			filepath.Join(xdgRuntimeDir, "app", "dev.vencord.Vesktop", "discord-ipc-%d"),
			filepath.Join(xdgRuntimeDir, ".flatpak", "dev.vencord.Vesktop", "xdg-run", "discord-ipc-%d"),
		)

		// 3. Official Discord Flatpak paths
		patterns = append(patterns,
			filepath.Join(xdgRuntimeDir, "app", "com.discordapp.Discord", "discord-ipc-%d"),
			filepath.Join(xdgRuntimeDir, ".flatpak", "com.discordapp.Discord", "xdg-run", "discord-ipc-%d"),
		)

		// 4. Discord Canary / PTB Flatpak
		patterns = append(patterns,
			filepath.Join(xdgRuntimeDir, "app", "com.discordapp.DiscordCanary", "discord-ipc-%d"),
			filepath.Join(xdgRuntimeDir, "app", "com.discordapp.DiscordPTB", "discord-ipc-%d"),
		)
	}

	if homeDir != "" {
		// Home-based Flatpak sandbox run paths
		patterns = append(patterns,
			filepath.Join(homeDir, ".var", "app", "dev.vencord.Vesktop", "xdg-run", "discord-ipc-%d"),
			filepath.Join(homeDir, ".var", "app", "com.discordapp.Discord", "xdg-run", "discord-ipc-%d"),
			filepath.Join(homeDir, ".var", "app", "com.discordapp.DiscordCanary", "xdg-run", "discord-ipc-%d"),
			filepath.Join(homeDir, ".var", "app", "com.discordapp.DiscordPTB", "xdg-run", "discord-ipc-%d"),
		)
	}

	// 5. Classic /tmp or macOS $TMPDIR
	patterns = append(patterns,
		filepath.Join(tmpDir, "discord-ipc-%d"),
		filepath.Join("/tmp", "discord-ipc-%d"),
	)

	return patterns
}

// Dial iterates through candidate socket locations from index 0 to 9, returning the first responsive connection.
func (d *UnixDialer) Dial(ctx context.Context) (net.Conn, error) {
	patterns := d.getSocketPatterns()
	dialer := net.Dialer{Timeout: d.DialTimeout}

	for _, pattern := range patterns {
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			sockPath := fmt.Sprintf(pattern, i)

			// Quickly check if path exists to avoid dial latency on non-existent files.
			if _, err := os.Stat(sockPath); err != nil {
				continue
			}

			conn, err := dialer.DialContext(ctx, "unix", sockPath)
			if err == nil {
				return conn, nil
			}
		}
	}

	return nil, ErrSocketNotFound
}
