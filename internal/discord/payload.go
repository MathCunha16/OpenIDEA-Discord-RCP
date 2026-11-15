package discord

import (
	"OpenIDEA-Discord-Rcp/pkg/discord"
)

// Re-export shared types from pkg/discord via aliases to maintain backwards compatibility.
type (
	Handshake        = discord.Handshake
	Timestamps       = discord.Timestamps
	Assets           = discord.Assets
	Button           = discord.Button
	Activity         = discord.Activity
	ActivityArgs     = discord.ActivityArgs
	ActivityCommand  = discord.ActivityCommand
	IncomingResponse = discord.IncomingResponse
	ErrorResponse    = discord.ErrorResponse
)
