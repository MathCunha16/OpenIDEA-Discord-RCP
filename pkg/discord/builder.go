package discord

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// FrameWriter abstracts writing framed bytes to a transport.
type FrameWriter interface {
	Write(p []byte) (n int, err error)
}

// FrameReader abstracts reading framed bytes from a transport.
type FrameReader interface {
	Read(p []byte) (n int, err error)
}

// BuildHandshakeFrame builds the contiguous 8-byte header + JSON body for OpHandshake (0).
func BuildHandshakeFrame(clientID string) ([]byte, error) {
	if clientID == "" {
		clientID = DefaultApplicationID
	}
	body, err := json.Marshal(Handshake{
		V:        1,
		ClientID: clientID,
	})
	if err != nil {
		return nil, fmt.Errorf("discord: marshal handshake: %w", err)
	}

	frame := make([]byte, HeaderSize+len(body))
	if err := writeContiguousHeader(frame, OpHandshake, uint32(len(body))); err != nil {
		return nil, err
	}
	copy(frame[HeaderSize:], body)
	return frame, nil
}

// BuildActivityFrame builds the contiguous 8-byte header + JSON body for OpFrame (1) SET_ACTIVITY.
func BuildActivityFrame(activity *Activity, pid int) ([]byte, error) {
	if pid <= 0 {
		pid = 1
	}
	nonce := fmt.Sprintf("%d", time.Now().UnixNano())
	cmd := ActivityCommand{
		Cmd: "SET_ACTIVITY",
		Args: ActivityArgs{
			Pid:      pid,
			Activity: activity,
		},
		Nonce: nonce,
	}

	body, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("discord: marshal activity: %w", err)
	}

	frame := make([]byte, HeaderSize+len(body))
	if err := writeContiguousHeader(frame, OpFrame, uint32(len(body))); err != nil {
		return nil, err
	}
	copy(frame[HeaderSize:], body)
	return frame, nil
}

// ParseIncomingFrame decodes a header and payload from a reader.
func ParseIncomingFrame(r io.Reader) (uint32, []byte, error) {
	return ReadFrame(r)
}

func writeContiguousHeader(buf []byte, op uint32, length uint32) error {
	if length > MaxPayloadSize {
		return ErrPayloadTooLarge
	}
	_ = buf[7] // bounds check hint
	buf[0] = byte(op)
	buf[1] = byte(op >> 8)
	buf[2] = byte(op >> 16)
	buf[3] = byte(op >> 24)
	buf[4] = byte(length)
	buf[5] = byte(length >> 8)
	buf[6] = byte(length >> 16)
	buf[7] = byte(length >> 24)
	return nil
}
