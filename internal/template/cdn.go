package template

import (
	"fmt"
	"strings"
)

// CDNBaseURL points to this repository's asset tree served via jsDelivr.
const CDNBaseURL = "https://cdn.jsdelivr.net/gh/MathCunha16/OpenIDEA-Discord-RCP@main/assets/icons"

// DefaultTheme specifies the fallback theme if none or an invalid one is supplied.
const DefaultTheme = "material"

// SupportedThemes lists the 6 icon themes provided in the assets directory.
var SupportedThemes = map[string]struct{}{
	"material":         {},
	"catppuccin_mocha": {},
	"catppuccin_latte": {},
	"classic":          {},
	"flat":             {},
	"atom":             {},
}

// NormalizeTheme ensures the theme name is recognized and valid.
func NormalizeTheme(theme string) string {
	clean := strings.ToLower(strings.TrimSpace(theme))
	if _, ok := SupportedThemes[clean]; ok {
		return clean
	}
	return DefaultTheme
}

// CDNResolver generates CDN URLs for language and application icons.
type CDNResolver struct {
	theme string
}

// NewCDNResolver initializes a resolver with the given theme.
func NewCDNResolver(theme string) *CDNResolver {
	return &CDNResolver{
		theme: NormalizeTheme(theme),
	}
}

// Theme returns the currently configured theme.
func (r *CDNResolver) Theme() string {
	return r.theme
}

// SetTheme updates the active theme.
func (r *CDNResolver) SetTheme(theme string) {
	r.theme = NormalizeTheme(theme)
}

// LanguageIconURL builds the complete CDN URL for a programming language or file type.
func (r *CDNResolver) LanguageIconURL(assetKey string) string {
	key := strings.ToLower(strings.TrimSpace(assetKey))
	if key == "" {
		key = "text"
	}
	return fmt.Sprintf("%s/themes/%s/languages/%s.png", CDNBaseURL, r.theme, key)
}

// ApplicationIconURL builds the complete CDN URL for an IDE application icon.
func (r *CDNResolver) ApplicationIconURL(appKey string) string {
	key := strings.ToLower(strings.TrimSpace(appKey))
	if key == "" {
		key = "openidea"
	}
	return fmt.Sprintf("%s/themes/%s/applications/%s.png", CDNBaseURL, r.theme, key)
}
