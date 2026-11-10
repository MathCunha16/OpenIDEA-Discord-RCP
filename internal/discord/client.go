package discord

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

var (
	// ErrNotConnected indicates an operation was attempted on an inactive connection.
	ErrNotConnected = errors.New("discord: client is not connected")
)

// Client defines the lifecycle and presence management interface for Discord IPC.
type Client interface {
	Connect(ctx context.Context) error
	SetActivity(activity *Activity) error
	ClearActivity() error
	Close() error
	IsConnected() bool
}

type client struct {
	clientID string
	dialer   SocketDialer

	mu        sync.Mutex
	conn      net.Conn
	connected bool
	pid       int
}

// ClientOptions allows custom configuration of the Discord client.
type ClientOptions struct {
	Dialer SocketDialer
	PID    int
}

// NewClient creates a new Discord IPC client instance with the given Application ID.
func NewClient(clientID string, opts ...func(*ClientOptions)) Client {
	options := ClientOptions{
		Dialer: DefaultDialer(),
		PID:    os.Getpid(),
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &client{
		clientID: clientID,
		dialer:   options.Dialer,
		pid:      options.PID,
	}
}

// Connect resolves the socket and completes the handshake protocol.
func (c *client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected && c.conn != nil {
		return nil
	}

	conn, err := c.dialer.Dial(ctx)
	if err != nil {
		return fmt.Errorf("discord: connection failed: %w", err)
	}

	// Send Handshake frame (Opcode 0)
	handshake := Handshake{
		V:        1,
		ClientID: c.clientID,
	}
	payload, err := json.Marshal(handshake)
	if err != nil {
		conn.Close()
		return fmt.Errorf("discord: handshake serialization error: %w", err)
	}

	if err := WriteFrame(conn, OpHandshake, payload); err != nil {
		conn.Close()
		return fmt.Errorf("discord: handshake write error: %w", err)
	}

	// Read handshake response
	opcode, respPayload, err := ReadFrame(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("discord: handshake read error: %w", err)
	}

	if opcode == OpClose {
		conn.Close()
		return fmt.Errorf("discord: connection rejected by discord: %s", string(respPayload))
	}

	c.conn = conn
	c.connected = true
	return nil
}

// SetActivity sends the SET_ACTIVITY command frame (Opcode 1) to Discord.
func (c *client) SetActivity(activity *Activity) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return ErrNotConnected
	}

	cmd := ActivityCommand{
		Cmd: "SET_ACTIVITY",
		Args: ActivityArgs{
			Pid:      c.pid,
			Activity: activity,
		},
		Nonce: generateNonce(),
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("discord: set activity serialization error: %w", err)
	}

	if err := WriteFrame(c.conn, OpFrame, payload); err != nil {
		// Mark connection as lost on write failure
		c.conn.Close()
		c.conn = nil
		c.connected = false
		return fmt.Errorf("discord: failed to send activity: %w", err)
	}

	return nil
}

// ClearActivity clears the user's presence card on Discord.
func (c *client) ClearActivity() error {
	return c.SetActivity(nil)
}

// Close gracefully terminates the connection to Discord IPC.
func (c *client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil
	}

	// Send OpClose notification best-effort
	_ = WriteFrame(c.conn, OpClose, []byte(`{}`))

	err := c.conn.Close()
	c.conn = nil
	c.connected = false
	return err
}

// IsConnected returns whether the client is currently connected.
func (c *client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// generateNonce creates a short unique hexadecimal string for frame correlation.
func generateNonce() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
