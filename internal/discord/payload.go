package discord

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
	Cmd   string `json:"cmd"`
	Evt   string `json:"evt,omitempty"`
	Nonce string `json:"nonce,omitempty"`
}
