package discord

import (
	"context"
	"net"
)

// SocketDialer defines the interface for establishing a connection to Discord's local IPC.
type SocketDialer interface {
	Dial(ctx context.Context) (net.Conn, error)
}
