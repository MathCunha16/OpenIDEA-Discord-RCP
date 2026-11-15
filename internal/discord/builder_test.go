package discord

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestBuildHandshakeFrame(t *testing.T) {
	frame, err := BuildHandshakeFrame("123456789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := bytes.NewReader(frame)
	op, payload, err := ReadFrame(r)
	if err != nil {
		t.Fatalf("ReadFrame error: %v", err)
	}
	if op != OpHandshake {
		t.Errorf("expected OpHandshake (0), got %d", op)
	}

	var hs Handshake
	if err := json.Unmarshal(payload, &hs); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if hs.ClientID != "123456789" || hs.V != 1 {
		t.Errorf("unexpected handshake payload: %+v", hs)
	}
}

func TestBuildActivityFrame(t *testing.T) {
	act := &Activity{
		Details: "Coding",
		State:   "main.go",
	}
	frame, err := BuildActivityFrame(act, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := bytes.NewReader(frame)
	op, payload, err := ReadFrame(r)
	if err != nil {
		t.Fatalf("ReadFrame error: %v", err)
	}
	if op != OpFrame {
		t.Errorf("expected OpFrame (1), got %d", op)
	}

	var cmd ActivityCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if cmd.Cmd != "SET_ACTIVITY" || cmd.Args.Pid != 42 || cmd.Args.Activity.Details != "Coding" {
		t.Errorf("unexpected activity command payload: %+v", cmd)
	}
}
