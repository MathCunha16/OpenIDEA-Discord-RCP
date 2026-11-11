package template

import (
	"fmt"
	"strings"
)

// CDNBaseURL points to this repository's asset tree served via raw GitHub content.
const CDNBaseURL = "https://raw.githubusercontent.com/MathCunha16/OpenIDEA-Discord-RCP/main/assets/icons"

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
	clean = strings.ReplaceAll(clean, " ", "_")
	if _, ok := SupportedThemes[clean]; ok {
		return clean
	}
	return DefaultTheme
}

// CDNResolver generates CDN URLs for language and application icons.
type CDNResolver struct {
	appTheme  string
	langTheme string
}

// NewCDNResolver initializes a resolver with the given themes.
func NewCDNResolver(appTheme, langTheme string) *CDNResolver {
	return &CDNResolver{
		appTheme:  NormalizeTheme(appTheme),
		langTheme: NormalizeTheme(langTheme),
	}
}

// AppTheme returns the configured application icon theme.
func (r *CDNResolver) AppTheme() string {
	return r.appTheme
}

// LangTheme returns the configured language icon theme.
func (r *CDNResolver) LangTheme() string {
	return r.langTheme
}

// SetThemes updates application and language icon themes.
func (r *CDNResolver) SetThemes(appTheme, langTheme string) {
	r.appTheme = NormalizeTheme(appTheme)
	r.langTheme = NormalizeTheme(langTheme)
}

// SetTheme updates both themes (backward compatibility).
func (r *CDNResolver) SetTheme(theme string) {
	norm := NormalizeTheme(theme)
	r.appTheme = norm
	r.langTheme = norm
}

// Theme returns the language theme.
func (r *CDNResolver) Theme() string {
	return r.langTheme
}

// hasIcon checks if the icon exists in the specified theme.
func hasIcon(theme, icon string) bool {
	if icons, ok := availableIcons[theme]; ok {
		_, exists := icons[icon]
		return exists
	}
	return false
}

// fallbackLanguageIcon returns the safest default icon for a theme.
func fallbackLanguageIcon(theme string) (string, string) {
	if hasIcon(theme, "text") {
		return theme, "text"
	}
	if hasIcon(theme, "binary") {
		return theme, "binary"
	}
	// Fall back to material theme text
	return "material", "text"
}

// LanguageIconURL builds the complete CDN URL for a programming language or file type.
func (r *CDNResolver) LanguageIconURL(assetKey string) string {
	key := strings.ToLower(strings.TrimSpace(assetKey))
	if key == "" || key == "text_file" {
		key = "text"
	}
	if key == "vb" {
		key = "visualbasic"
	}

	// 1. Direct check in current theme
	if hasIcon(r.langTheme, key) {
		return fmt.Sprintf("%s/themes/%s/languages/%s.png", CDNBaseURL, r.langTheme, key)
	}

	// 2. Check semantic aliases
	alias := ""
	switch key {
	case "shell", "bash", "zsh", "powershell":
		alias = "terminal"
	case "sql":
		alias = "database"
	case "wasm", "wat", "wit":
		alias = "binary"
	case "protobuf", "proto":
		alias = "config"
	case "svelte", "astro":
		alias = "html"
	case "zig", "odin":
		alias = "c"
	case "react":
		alias = "reactjs"
	case "dockerfile":
		alias = "docker"
	case "rust":
		alias = "cargo"
	}

	if alias != "" && hasIcon(r.langTheme, alias) {
		return fmt.Sprintf("%s/themes/%s/languages/%s.png", CDNBaseURL, r.langTheme, alias)
	}

	// 3. Fallback to material theme if material has the exact icon
	if r.langTheme != "material" && hasIcon("material", key) {
		return fmt.Sprintf("%s/themes/material/languages/%s.png", CDNBaseURL, key)
	}

	// 4. Fallback to theme default (text or binary)
	fbTheme, fbIcon := fallbackLanguageIcon(r.langTheme)
	return fmt.Sprintf("%s/themes/%s/languages/%s.png", CDNBaseURL, fbTheme, fbIcon)
}

// ApplicationIconURL builds the complete CDN URL for an IDE application icon.
func (r *CDNResolver) ApplicationIconURL(appKey string) string {
	key := strings.ToLower(strings.TrimSpace(appKey))
	if key == "" {
		key = "openidea"
	}
	return fmt.Sprintf("%s/themes/%s/applications/%s.png", CDNBaseURL, r.appTheme, key)
}
