package discord

import (
	"encoding/json"
	"errors"
)

// Discord IPC Opcode definitions.
const (
	OpHandshake uint32 = 0
	OpFrame     uint32 = 1
	OpClose     uint32 = 2
	OpPing      uint32 = 3
	OpPong      uint32 = 4
)

// HeaderSize represents the 8-byte little-endian header length (4 bytes opcode + 4 bytes length).
const HeaderSize = 8

// MaxPayloadSize protects against abnormal or corrupt frame lengths (1 MB limit).
const MaxPayloadSize = 1024 * 1024

// DefaultApplicationID is the official Discord application client ID for OpenIDEA.
const DefaultApplicationID = "1569761142878896160"

var (
	// ErrPayloadTooLarge indicates the incoming frame exceeds MaxPayloadSize.
	ErrPayloadTooLarge = errors.New("discord: payload exceeds maximum allowable size")
)

// Handshake represents the initial connection message sent to Discord IPC.
type Handshake struct {
	V        int    `json:"v"`
	ClientID string `json:"client_id"`
}

// Timestamps defines start and end epoch timestamps in seconds.
type Timestamps struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

// Assets defines rich display images and tooltip strings.
type Assets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

// Button defines an external clickable link inside the presence card.
type Button struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// Activity represents the Discord Rich Presence activity structure.
type Activity struct {
	Details    string      `json:"details,omitempty"`
	State      string      `json:"state,omitempty"`
	Timestamps *Timestamps `json:"timestamps,omitempty"`
	Assets     *Assets     `json:"assets,omitempty"`
	Buttons    []Button    `json:"buttons,omitempty"`
}

// Equal returns true if two Activity objects are structurally identical.
func (a *Activity) Equal(other *Activity) bool {
	if a == nil && other == nil {
		return true
	}
	if a == nil || other == nil {
		return false
	}
	if a.Details != other.Details || a.State != other.State {
		return false
	}
	if (a.Timestamps == nil) != (other.Timestamps == nil) {
		return false
	}
	if a.Timestamps != nil {
		if a.Timestamps.Start != other.Timestamps.Start || a.Timestamps.End != other.Timestamps.End {
			return false
		}
	}
	if (a.Assets == nil) != (other.Assets == nil) {
		return false
	}
	if a.Assets != nil {
		if a.Assets.LargeImage != other.Assets.LargeImage ||
			a.Assets.LargeText != other.Assets.LargeText ||
			a.Assets.SmallImage != other.Assets.SmallImage ||
			a.Assets.SmallText != other.Assets.SmallText {
			return false
		}
	}
	if len(a.Buttons) != len(other.Buttons) {
		return false
	}
	for i := range a.Buttons {
		if a.Buttons[i].Label != other.Buttons[i].Label || a.Buttons[i].URL != other.Buttons[i].URL {
			return false
		}
	}
	return true
}

// ActivityArgs is the wrapper payload required by the SET_ACTIVITY command.
type ActivityArgs struct {
	Pid      int       `json:"pid"`
	Activity *Activity `json:"activity,omitempty"`
}

// ActivityCommand represents the full envelope sent to Discord IPC over Opcode 1.
type ActivityCommand struct {
	Cmd   string       `json:"cmd"`
	Args  ActivityArgs `json:"args"`
	Nonce string       `json:"nonce"`
}

// IncomingResponse represents a generic response or event received from Discord IPC.
type IncomingResponse struct {
	Cmd   string          `json:"cmd"`
	Data  json.RawMessage `json:"data,omitempty"`
	Evt   string          `json:"evt,omitempty"`
	Nonce string          `json:"nonce,omitempty"`
}

// ErrorResponse represents an error payload returned by Discord when evt == "ERROR".
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
