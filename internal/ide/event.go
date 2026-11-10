package ide

import (
	"encoding/json"
	"io"
)

// EditorEvent represents an incoming event emitted by OpenIDEA via stdio.
type EditorEvent struct {
	// Event type (e.g. "active-file-changed", "editor-focused", "idle", "shutdown").
	Type string `json:"type,omitempty"`

	// File path currently focused or edited (relative or absolute).
	File string `json:"file,omitempty"`

	// Language ID as reported by OpenIDEA/LSP (optional, e.g. "go", "python").
	Language string `json:"language,omitempty"`

	// Name of the open project/workspace.
	Project string `json:"project,omitempty"`

	// Active VCS Git branch (e.g. "main", "feature/login").
	Branch string `json:"branch,omitempty"`

	// Current cursor line index (optional, 1-indexed).
	CursorLine int `json:"cursor_line,omitempty"`

	// Total lines in current buffer (optional).
	TotalLines int `json:"total_lines,omitempty"`

	// PrivacyMode toggle (if true, hides specific file names).
	PrivacyMode bool `json:"privacy_mode,omitempty"`
}

// EventReader provides high-performance line-delimited JSON stream decoding for EditorEvents.
type EventReader struct {
	decoder *json.Decoder
}

// NewEventReader creates an EventReader bound to an io.Reader (typically os.Stdin).
func NewEventReader(r io.Reader) *EventReader {
	return &EventReader{
		decoder: json.NewDecoder(r),
	}
}

// ReadNext decodes the next EditorEvent from the stream.
func (r *EventReader) ReadNext() (*EditorEvent, error) {
	var ev EditorEvent
	if err := r.decoder.Decode(&ev); err != nil {
		return nil, err
	}
	return &ev, nil
}
