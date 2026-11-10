package discord

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

type mockDialer struct {
	server net.Conn
	client net.Conn
}

func newMockDialer() *mockDialer {
	server, client := net.Pipe()
	return &mockDialer{
		server: server,
		client: client,
	}
}

func (m *mockDialer) Dial(ctx context.Context) (net.Conn, error) {
	return m.client, nil
}

func TestClientLifecycleAndSetActivity(t *testing.T) {
	dialer := newMockDialer()
	cli := NewClient("1569761142878896160", func(opts *ClientOptions) {
		opts.Dialer = dialer
		opts.PID = 4321
	})

	serverDone := make(chan struct{})

	// Emulate Discord IPC server
	go func() {
		defer close(serverDone)
		defer dialer.server.Close()

		// 1. Expect OpHandshake
		opcode, payload, err := ReadFrame(dialer.server)
		if err != nil {
			t.Errorf("server read handshake error: %v", err)
			return
		}
		if opcode != OpHandshake {
			t.Errorf("expected OpHandshake (0), got %d", opcode)
			return
		}

		var hs Handshake
		if err := json.Unmarshal(payload, &hs); err != nil {
			t.Errorf("server unmarshal handshake error: %v", err)
			return
		}
		if hs.ClientID != "1569761142878896160" {
			t.Errorf("expected client id 1569761142878896160, got %s", hs.ClientID)
			return
		}

		// Reply to Handshake with READY frame
		readyResp := []byte(`{"cmd":"DISPATCH","evt":"READY","data":{"v":1}}`)
		if err := WriteFrame(dialer.server, OpFrame, readyResp); err != nil {
			t.Errorf("server write ready error: %v", err)
			return
		}

		// 2. Expect SET_ACTIVITY frame
		opcode, payload, err = ReadFrame(dialer.server)
		if err != nil {
			t.Errorf("server read activity error: %v", err)
			return
		}
		if opcode != OpFrame {
			t.Errorf("expected OpFrame (1), got %d", opcode)
			return
		}

		var cmd ActivityCommand
		if err := json.Unmarshal(payload, &cmd); err != nil {
			t.Errorf("server unmarshal activity error: %v", err)
			return
		}

		if cmd.Cmd != "SET_ACTIVITY" {
			t.Errorf("expected cmd SET_ACTIVITY, got %s", cmd.Cmd)
		}
		if cmd.Args.Activity.Details != "Editing main.go" {
			t.Errorf("expected details 'Editing main.go', got '%s'", cmd.Args.Activity.Details)
		}

		// 3. Expect OpClose
		opcode, _, err = ReadFrame(dialer.server)
		if err != nil {
			return
		}
		if opcode != OpClose {
			t.Errorf("expected OpClose (2), got %d", opcode)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := cli.Connect(ctx); err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}

	if !cli.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	activity := &Activity{
		Details: "Editing main.go",
		State:   "Project: OpenIDEA-Discord-Rcp",
		Timestamps: &Timestamps{
			Start: 123456789,
		},
		Assets: &Assets{
			LargeImage: "go_logo",
			LargeText:  "Go 1.26",
		},
	}

	if err := cli.SetActivity(activity); err != nil {
		t.Fatalf("client.SetActivity failed: %v", err)
	}

	if err := cli.Close(); err != nil {
		t.Fatalf("client.Close failed: %v", err)
	}

	<-serverDone
}

func TestClientNotConnected(t *testing.T) {
	cli := NewClient("123456789")
	err := cli.SetActivity(&Activity{Details: "test"})
	if err != ErrNotConnected {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}
