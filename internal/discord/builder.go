package discord

import (
	"io"

	"OpenIDEA-Discord-Rcp/pkg/discord"
)

// DefaultApplicationID is the official Discord application client ID for OpenIDEA.
const DefaultApplicationID = discord.DefaultApplicationID

// FrameWriter abstracts writing framed bytes to a transport.
type FrameWriter = discord.FrameWriter

// FrameReader abstracts reading framed bytes from a transport.
type FrameReader = discord.FrameReader

// BuildHandshakeFrame builds the contiguous 8-byte header + JSON body for OpHandshake (0).
func BuildHandshakeFrame(clientID string) ([]byte, error) {
	return discord.BuildHandshakeFrame(clientID)
}

// BuildActivityFrame builds the contiguous 8-byte header + JSON body for OpFrame (1) SET_ACTIVITY.
func BuildActivityFrame(activity *Activity, pid int) ([]byte, error) {
	return discord.BuildActivityFrame(activity, pid)
}

// BuildPongFrame builds the contiguous 8-byte header + echo body for OpPong (4).
func BuildPongFrame(payload []byte) ([]byte, error) {
	return discord.BuildPongFrame(payload)
}

// ParseIncomingFrame decodes a header and payload from a reader.
func ParseIncomingFrame(r io.Reader) (uint32, []byte, error) {
	return discord.ParseIncomingFrame(r)
}
