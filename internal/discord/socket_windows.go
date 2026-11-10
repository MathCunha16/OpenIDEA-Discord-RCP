//go:build windows

package discord

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

// ErrSocketNotFound indicates that no responsive Discord Named Pipe was discovered on Windows.
var ErrSocketNotFound = errors.New("discord: no active discord named pipe found")

// WindowsDialer connects to Discord via Windows Named Pipes (\\.\pipe\discord-ipc-[0-9]).
type WindowsDialer struct {
	Timeout time.Duration
}

// NewWindowsDialer creates a WindowsDialer with standard timeout defaults.
func NewWindowsDialer() *WindowsDialer {
	return &WindowsDialer{
		Timeout: 500 * time.Millisecond,
	}
}

// Dial iterates through \\.\pipe\discord-ipc-0 to 9, returning the first connected pipe.
func (d *WindowsDialer) Dial(ctx context.Context) (net.Conn, error) {
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		pipePath := fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i)
		dialCtx, cancel := context.WithTimeout(ctx, d.Timeout)
		conn, err := winio.DialPipeContext(dialCtx, pipePath)
		cancel()
		if err == nil {
			return conn, nil
		}
	}

	return nil, ErrSocketNotFound
}
