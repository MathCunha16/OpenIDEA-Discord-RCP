package discord

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestWriteFrameAndReadFrame(t *testing.T) {
	buf := new(bytes.Buffer)
	payload := []byte(`{"cmd":"SET_ACTIVITY","data":123}`)

	if err := WriteFrame(buf, OpFrame, payload); err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	if buf.Len() != HeaderSize+len(payload) {
		t.Fatalf("expected written length %d, got %d", HeaderSize+len(payload), buf.Len())
	}

	opcode, readPayload, err := ReadFrame(buf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	if opcode != OpFrame {
		t.Fatalf("expected opcode %d, got %d", OpFrame, opcode)
	}

	if !bytes.Equal(payload, readPayload) {
		t.Fatalf("expected payload %s, got %s", payload, readPayload)
	}
}

func TestWriteFrameEmptyPayload(t *testing.T) {
	buf := new(bytes.Buffer)

	if err := WriteFrame(buf, OpPing, nil); err != nil {
		t.Fatalf("WriteFrame with empty payload failed: %v", err)
	}

	opcode, readPayload, err := ReadFrame(buf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	if opcode != OpPing {
		t.Fatalf("expected opcode %d, got %d", OpPing, opcode)
	}

	if len(readPayload) != 0 {
		t.Fatalf("expected 0 bytes payload, got %d", len(readPayload))
	}
}

func TestReadFramePayloadTooLarge(t *testing.T) {
	buf := new(bytes.Buffer)
	var header [HeaderSize]byte
	binary.LittleEndian.PutUint32(header[0:4], OpFrame)
	binary.LittleEndian.PutUint32(header[4:8], MaxPayloadSize+10)

	buf.Write(header[:])

	_, _, err := ReadFrame(buf)
	if err != ErrPayloadTooLarge {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}
}

func TestReadFrameTruncatedHeader(t *testing.T) {
	buf := bytes.NewReader([]byte{0x00, 0x01})
	_, _, err := ReadFrame(buf)
	if err == nil || err == io.EOF {
		t.Fatalf("expected unexpected EOF or read error, got %v", err)
	}
}
