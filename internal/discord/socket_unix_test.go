//go:build !windows

package discord

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUnixDialerConnectSuccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "discord-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sockPath := filepath.Join(tempDir, "discord-ipc-2")
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to listen on test socket: %v", err)
	}
	defer listener.Close()

	pattern := filepath.Join(tempDir, "discord-ipc-%d")
	dialer := &UnixDialer{
		CustomPatterns: []string{pattern},
		DialTimeout:    200 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := dialer.Dial(ctx)
	if err != nil {
		t.Fatalf("expected dialer to succeed, got error: %v", err)
	}
	defer conn.Close()

	if conn.RemoteAddr().String() != sockPath {
		t.Fatalf("expected remote addr %s, got %s", sockPath, conn.RemoteAddr().String())
	}
}

func TestUnixDialerNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "discord-test-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pattern := filepath.Join(tempDir, "nonexistent-ipc-%d")
	dialer := &UnixDialer{
		CustomPatterns: []string{pattern},
		DialTimeout:    50 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err = dialer.Dial(ctx)
	if err != ErrSocketNotFound {
		t.Fatalf("expected ErrSocketNotFound, got %v", err)
	}
}
