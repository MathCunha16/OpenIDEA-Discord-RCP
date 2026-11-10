package template

import (
	"strings"
	"unicode/utf8"
)

// Discord text length constraints.
const (
	MinDiscordLength = 2
	MaxDiscordLength = 128

	// EmptyLineHack uses the zero-width space Unicode character (\u200B)
	// to satisfy Discord IPC's 2-character minimum without visual artifacts.
	ZeroWidthSpace = "\u200B"
	Ellipsis       = "..."
)

// FitToRange enforces Discord Rich Presence text field constraints [min, max].
// - If the string is empty or blank, it returns an empty string or padded empty line hack.
// - If length < min, it pads the string with ZeroWidthSpace (\u200B).
// - If length > max, it smoothly truncates with an ellipsis ("...").
func FitToRange(s string, min, max int) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return strings.Repeat(ZeroWidthSpace, min)
	}

	runeCount := utf8.RuneCountInString(trimmed)

	if runeCount < min {
		return trimmed + strings.Repeat(ZeroWidthSpace, min-runeCount)
	}

	if runeCount > max {
		// Truncate cleanly on rune boundary, reserving 3 characters for ellipsis
		runes := []rune(trimmed)
		cutoff := max - utf8.RuneCountInString(Ellipsis)
		if cutoff < 0 {
			cutoff = 0
		}
		return string(runes[:cutoff]) + Ellipsis
	}

	return trimmed
}

// FitDiscordString applies default Discord limits (2 to 128 characters).
func FitDiscordString(s string) string {
	return FitToRange(s, MinDiscordLength, MaxDiscordLength)
}

// IsInvisible checks whether the string consists solely of whitespace or zero-width spaces.
func IsInvisible(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' && r != '\u200B' {
			return false
		}
	}
	return true
}
