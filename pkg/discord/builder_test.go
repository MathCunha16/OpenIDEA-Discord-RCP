package discord

import (
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestBuildHandshakeFrame(t *testing.T) {
	frame, err := BuildHandshakeFrame("123456789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(frame) <= HeaderSize {
		t.Fatalf("frame too short: %d", len(frame))
	}

	op := binary.LittleEndian.Uint32(frame[0:4])
	if op != OpHandshake {
		t.Errorf("expected op %d, got %d", OpHandshake, op)
	}

	payloadLen := binary.LittleEndian.Uint32(frame[4:8])
	if int(payloadLen) != len(frame)-HeaderSize {
		t.Errorf("expected payload length %d, got %d", len(frame)-HeaderSize, payloadLen)
	}

	var hs Handshake
	if err := json.Unmarshal(frame[HeaderSize:], &hs); err != nil {
		t.Fatalf("failed to parse handshake payload: %v", err)
	}
	if hs.ClientID != "123456789" || hs.V != 1 {
		t.Errorf("handshake payload mismatch: %+v", hs)
	}
}

func TestBuildActivityFrame(t *testing.T) {
	act := &Activity{
		Details: "Editing main.go",
		State:   "Workspace: test",
	}

	frame, err := BuildActivityFrame(act, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	op := binary.LittleEndian.Uint32(frame[0:4])
	if op != OpFrame {
		t.Errorf("expected op %d, got %d", OpFrame, op)
	}

	var cmd ActivityCommand
	if err := json.Unmarshal(frame[HeaderSize:], &cmd); err != nil {
		t.Fatalf("failed to parse activity command: %v", err)
	}

	if cmd.Cmd != "SET_ACTIVITY" || cmd.Args.Pid != 42 || cmd.Args.Activity.Details != "Editing main.go" {
		t.Errorf("activity command mismatch: %+v", cmd)
	}
}

func TestBuildPongFrame(t *testing.T) {
	echo := []byte(`{"ping":123}`)
	frame, err := BuildPongFrame(echo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	op := binary.LittleEndian.Uint32(frame[0:4])
	if op != OpPong {
		t.Errorf("expected op %d, got %d", OpPong, op)
	}

	payloadLen := binary.LittleEndian.Uint32(frame[4:8])
	if int(payloadLen) != len(echo) {
		t.Errorf("expected payload length %d, got %d", len(echo), payloadLen)
	}

	if string(frame[HeaderSize:]) != string(echo) {
		t.Errorf("expected payload %s, got %s", echo, frame[HeaderSize:])
	}
}
