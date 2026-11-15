package discord

import (
	"encoding/binary"
	"fmt"
	"io"
)

// WriteFrame encodes and transmits a Discord IPC frame using an 8-byte little-endian header.
// It performs a single contiguous write to minimize syscalls and prevent partial writes.
func WriteFrame(w io.Writer, opcode uint32, payload []byte) error {
	payloadLen := len(payload)
	if payloadLen > MaxPayloadSize {
		return ErrPayloadTooLarge
	}

	frame := make([]byte, HeaderSize+payloadLen)
	binary.LittleEndian.PutUint32(frame[0:4], opcode)
	binary.LittleEndian.PutUint32(frame[4:8], uint32(payloadLen))

	if payloadLen > 0 {
		copy(frame[HeaderSize:], payload)
	}

	_, err := w.Write(frame)
	if err != nil {
		return fmt.Errorf("discord: write frame error: %w", err)
	}
	return nil
}

// ReadFrame decodes a Discord IPC frame from an io.Reader.
// It strictly parses the 8-byte little-endian header and reads the exact payload bytes.
func ReadFrame(r io.Reader) (uint32, []byte, error) {
	var header [HeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, nil, fmt.Errorf("discord: read header error: %w", err)
	}

	opcode := binary.LittleEndian.Uint32(header[0:4])
	length := binary.LittleEndian.Uint32(header[4:8])

	if length > MaxPayloadSize {
		return 0, nil, ErrPayloadTooLarge
	}

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, nil, fmt.Errorf("discord: read payload error: %w", err)
		}
	}

	return opcode, payload, nil
}
