package discord

import (
	"OpenIDEA-Discord-Rcp/pkg/discord"
)

// Discord IPC Opcode definitions re-exported from pkg/discord.
const (
	OpHandshake = discord.OpHandshake
	OpFrame     = discord.OpFrame
	OpClose     = discord.OpClose
	OpPing      = discord.OpPing
	OpPong      = discord.OpPong
)

// Header and size constants re-exported from pkg/discord.
const (
	HeaderSize     = discord.HeaderSize
	MaxPayloadSize = discord.MaxPayloadSize
)

var (
	// ErrPayloadTooLarge re-exported from pkg/discord.
	ErrPayloadTooLarge = discord.ErrPayloadTooLarge
)

var (
	// WriteFrame re-exported from pkg/discord.
	WriteFrame = discord.WriteFrame
	// ReadFrame re-exported from pkg/discord.
	ReadFrame = discord.ReadFrame
)
