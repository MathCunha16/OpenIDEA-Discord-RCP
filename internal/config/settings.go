package config

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"OpenIDEA-Discord-Rcp/internal/tracker"
)

// Default settings values.
const (
	DefaultApplicationID = "1569761142878896160"
	DefaultAppTheme      = "material"
	DefaultLangTheme     = "classic"
	DefaultDebounceMs    = 1500
	DefaultIdleMinutes   = 5
)

// Settings represents the configurable preferences for OpenIDEA Discord Rich Presence.
type Settings struct {
	Enabled               bool   `json:"enabled"`
	ApplicationID         string `json:"application_id"`
	ApplicationTheme      string `json:"application_theme"`
	LanguageTheme         string `json:"language_theme"`
	Theme                 string `json:"theme"` // Backward compatibility fallback
	DebounceMs            int    `json:"debounce_ms"`
	DetectIdle            bool   `json:"detect_idle"`
	IdleTimeoutMins       int    `json:"idle_timeout_mins"`
	WhileIdle             string `json:"while_idle"` // "keep_showing", "show_idling", "hide_presence"
	ResetOpenTimeOnReturn bool   `json:"reset_open_time_on_return"`
	TimeMode              string `json:"time_mode"` // "project", "file", "none"
	ShowGitButton         bool   `json:"show_git_button"`
	ShowIDEButton         bool   `json:"show_ide_button"`
	PrivacyMode           bool   `json:"privacy_mode"`

	// Layout: File
	FileFirstLine        string `json:"file_first_line"`
	FileSecondLine       string `json:"file_second_line"`
	PrefixReadingEditing bool   `json:"prefix_reading_editing"`
	FileLargeIconRole    string `json:"file_large_icon_role"` // "language", "application"
	FileSmallIconRole    string `json:"file_small_icon_role"` // "application", "none"
	FileTimeMode         string `json:"file_time_mode"`       // "file", "project", "none"

	// Layout: Project
	ProjectFirstLine  string `json:"project_first_line"`
	ProjectSecondLine string `json:"project_second_line"`
	ProjectTimeMode   string `json:"project_time_mode"` // "project", "none"
}

// DefaultSettings returns the standard configuration for the daemon.
func DefaultSettings() Settings {
	return Settings{
		Enabled:               true,
		ApplicationID:         DefaultApplicationID,
		ApplicationTheme:      DefaultAppTheme,
		LanguageTheme:         DefaultLangTheme,
		Theme:                 DefaultAppTheme,
		DebounceMs:            DefaultDebounceMs,
		DetectIdle:            true,
		IdleTimeoutMins:       DefaultIdleMinutes,
		WhileIdle:             "show_idling",
		ResetOpenTimeOnReturn: false,
		TimeMode:              "project",
		ShowGitButton:         true,
		ShowIDEButton:         true,
		PrivacyMode:           false,
		FileFirstLine:         "{project} ({branch})",
		FileSecondLine:        "Editing {file}",
		PrefixReadingEditing:  true,
		FileLargeIconRole:     "language",
		FileSmallIconRole:     "application",
		FileTimeMode:          "project",
		ProjectFirstLine:      "{project} ({branch})",
		ProjectSecondLine:     "Navigating project",
		ProjectTimeMode:       "project",
	}
}

// UnmarshalJSON supports decoding both camelCase and snake_case configurations.
func (s *Settings) UnmarshalJSON(data []byte) error {
	type RawSettings struct {
		Enabled                   *bool  `json:"enabled"`
		ApplicationID             string `json:"application_id"`
		ApplicationIDCamel        string `json:"applicationId"`
		ApplicationTheme          string `json:"application_theme"`
		ApplicationThemeCamel     string `json:"applicationTheme"`
		LanguageTheme             string `json:"language_theme"`
		LanguageThemeCamel        string `json:"languageTheme"`
		Theme                     string `json:"theme"`
		DebounceMs                *int   `json:"debounce_ms"`
		DebounceMsCamel           *int   `json:"debounceMs"`
		DetectIdle                *bool  `json:"detect_idle"`
		DetectIdleCamel           *bool  `json:"detectIdle"`
		IdleTimeoutMins           *int   `json:"idle_timeout_mins"`
		IdleTimeoutMinsCamel      *int   `json:"idleTimeoutMins"`
		WhileIdle                 string `json:"while_idle"`
		WhileIdleCamel            string `json:"whileIdle"`
		ResetOpenTimeOnReturn     *bool  `json:"reset_open_time_on_return"`
		ResetOpenTimeOnReturnCamel *bool `json:"resetOpenTimeOnReturn"`
		TimeMode                  string `json:"time_mode"`
		TimeModeCamel             string `json:"timeMode"`
		ShowGitButton             *bool  `json:"show_git_button"`
		ShowGitButtonCamel        *bool  `json:"showGitButton"`
		ShowIDEButton             *bool  `json:"show_ide_button"`
		ShowIDEButtonCamel        *bool  `json:"showIDEButton"`
		PrivacyMode               *bool  `json:"privacy_mode"`
		PrivacyModeCamel          *bool  `json:"privacyMode"`

		// File layout
		FileFirstLine              string `json:"file_first_line"`
		FileFirstLineCamel         string `json:"fileFirstLine"`
		FileSecondLine             string `json:"file_second_line"`
		FileSecondLineCamel        string `json:"fileSecondLine"`
		PrefixReadingEditing       *bool  `json:"prefix_reading_editing"`
		PrefixReadingEditingCamel  *bool  `json:"prefixReadingEditing"`
		FileLargeIconRole          string `json:"file_large_icon_role"`
		FileLargeIconRoleCamel     string `json:"fileLargeIconRole"`
		FileSmallIconRole          string `json:"file_small_icon_role"`
		FileSmallIconRoleCamel     string `json:"fileSmallIconRole"`
		FileTimeMode               string `json:"file_time_mode"`
		FileTimeModeCamel          string `json:"fileTimeMode"`

		// Project layout
		ProjectFirstLine          string `json:"project_first_line"`
		ProjectFirstLineCamel     string `json:"projectFirstLine"`
		ProjectSecondLine         string `json:"project_second_line"`
		ProjectSecondLineCamel    string `json:"projectSecondLine"`
		ProjectTimeMode           string `json:"project_time_mode"`
		ProjectTimeModeCamel      string `json:"projectTimeMode"`
	}

	var raw RawSettings
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.Enabled != nil {
		s.Enabled = *raw.Enabled
	}

	if raw.ApplicationID != "" {
		s.ApplicationID = raw.ApplicationID
	} else if raw.ApplicationIDCamel != "" {
		s.ApplicationID = raw.ApplicationIDCamel
	}

	if raw.ApplicationTheme != "" {
		s.ApplicationTheme = raw.ApplicationTheme
	} else if raw.ApplicationThemeCamel != "" {
		s.ApplicationTheme = raw.ApplicationThemeCamel
	}

	if raw.LanguageTheme != "" {
		s.LanguageTheme = raw.LanguageTheme
	} else if raw.LanguageThemeCamel != "" {
		s.LanguageTheme = raw.LanguageThemeCamel
	}

	if raw.Theme != "" {
		s.Theme = raw.Theme
		if raw.ApplicationTheme == "" && raw.ApplicationThemeCamel == "" {
			s.ApplicationTheme = raw.Theme
		}
		if raw.LanguageTheme == "" && raw.LanguageThemeCamel == "" {
			s.LanguageTheme = raw.Theme
		}
	}

	if raw.DebounceMs != nil {
		s.DebounceMs = *raw.DebounceMs
	} else if raw.DebounceMsCamel != nil {
		s.DebounceMs = *raw.DebounceMsCamel
	}

	if raw.DetectIdle != nil {
		s.DetectIdle = *raw.DetectIdle
	} else if raw.DetectIdleCamel != nil {
		s.DetectIdle = *raw.DetectIdleCamel
	}

	if raw.IdleTimeoutMins != nil {
		s.IdleTimeoutMins = *raw.IdleTimeoutMins
	} else if raw.IdleTimeoutMinsCamel != nil {
		s.IdleTimeoutMins = *raw.IdleTimeoutMinsCamel
	}

	if raw.WhileIdle != "" {
		s.WhileIdle = raw.WhileIdle
	} else if raw.WhileIdleCamel != "" {
		s.WhileIdle = raw.WhileIdleCamel
	}

	if raw.ResetOpenTimeOnReturn != nil {
		s.ResetOpenTimeOnReturn = *raw.ResetOpenTimeOnReturn
	} else if raw.ResetOpenTimeOnReturnCamel != nil {
		s.ResetOpenTimeOnReturn = *raw.ResetOpenTimeOnReturnCamel
	}

	if raw.TimeMode != "" {
		s.TimeMode = raw.TimeMode
	} else if raw.TimeModeCamel != "" {
		s.TimeMode = raw.TimeModeCamel
	}

	if raw.ShowGitButton != nil {
		s.ShowGitButton = *raw.ShowGitButton
	} else if raw.ShowGitButtonCamel != nil {
		s.ShowGitButton = *raw.ShowGitButtonCamel
	}

	if raw.ShowIDEButton != nil {
		s.ShowIDEButton = *raw.ShowIDEButton
	} else if raw.ShowIDEButtonCamel != nil {
		s.ShowIDEButton = *raw.ShowIDEButtonCamel
	}

	if raw.PrivacyMode != nil {
		s.PrivacyMode = *raw.PrivacyMode
	} else if raw.PrivacyModeCamel != nil {
		s.PrivacyMode = *raw.PrivacyModeCamel
	}

	// File layout
	if raw.FileFirstLine != "" {
		s.FileFirstLine = raw.FileFirstLine
	} else if raw.FileFirstLineCamel != "" {
		s.FileFirstLine = raw.FileFirstLineCamel
	}

	if raw.FileSecondLine != "" {
		s.FileSecondLine = raw.FileSecondLine
	} else if raw.FileSecondLineCamel != "" {
		s.FileSecondLine = raw.FileSecondLineCamel
	}

	if raw.PrefixReadingEditing != nil {
		s.PrefixReadingEditing = *raw.PrefixReadingEditing
	} else if raw.PrefixReadingEditingCamel != nil {
		s.PrefixReadingEditing = *raw.PrefixReadingEditingCamel
	}

	if raw.FileLargeIconRole != "" {
		s.FileLargeIconRole = raw.FileLargeIconRole
	} else if raw.FileLargeIconRoleCamel != "" {
		s.FileLargeIconRole = raw.FileLargeIconRoleCamel
	}

	if raw.FileSmallIconRole != "" {
		s.FileSmallIconRole = raw.FileSmallIconRole
	} else if raw.FileSmallIconRoleCamel != "" {
		s.FileSmallIconRole = raw.FileSmallIconRoleCamel
	}

	if raw.FileTimeMode != "" {
		s.FileTimeMode = raw.FileTimeMode
	} else if raw.FileTimeModeCamel != "" {
		s.FileTimeMode = raw.FileTimeModeCamel
	}

	// Project layout
	if raw.ProjectFirstLine != "" {
		s.ProjectFirstLine = raw.ProjectFirstLine
	} else if raw.ProjectFirstLineCamel != "" {
		s.ProjectFirstLine = raw.ProjectFirstLineCamel
	}

	if raw.ProjectSecondLine != "" {
		s.ProjectSecondLine = raw.ProjectSecondLine
	} else if raw.ProjectSecondLineCamel != "" {
		s.ProjectSecondLine = raw.ProjectSecondLineCamel
	}

	if raw.ProjectTimeMode != "" {
		s.ProjectTimeMode = raw.ProjectTimeMode
	} else if raw.ProjectTimeModeCamel != "" {
		s.ProjectTimeMode = raw.ProjectTimeModeCamel
	}

	s.NormalizePresets()
	return nil
}

// NormalizePresets converts human-friendly UI labels (from IDE settings dropdowns) into internal template strings.
func (s *Settings) NormalizePresets() {
	// WhileIdle
	switch strings.ToLower(s.WhileIdle) {
	case "show idling", "show idling status", "show_idling":
		s.WhileIdle = "show_idling"
	case "keep current status", "keep showing rich presence", "keep_showing":
		s.WhileIdle = "keep_showing"
	case "hide presence", "hide presence completely", "hide_presence":
		s.WhileIdle = "hide_presence"
	}

	// FileFirstLine
	switch s.FileFirstLine {
	case "Project Name (Branch)", "Project Name - VCS branch":
		s.FileFirstLine = "{project} ({branch})"
	case "Project Name only":
		s.FileFirstLine = "{project}"
	case "File Name":
		s.FileFirstLine = "{file}"
	case "Empty":
		s.FileFirstLine = ""
	}

	// FileSecondLine
	switch s.FileSecondLine {
	case "Editing file", "File Name (Reading / Editing)":
		s.FileSecondLine = "Editing {file}"
	case "File name only", "File Name only":
		s.FileSecondLine = "{file}"
	case "Project name", "Project Name":
		s.FileSecondLine = "{project}"
	case "Empty":
		s.FileSecondLine = ""
	}

	// FileLargeIconRole
	switch s.FileLargeIconRole {
	case "File Language Icon", "language":
		s.FileLargeIconRole = "language"
	case "Application (OpenIDEA) Icon", "application":
		s.FileLargeIconRole = "application"
	}

	// FileSmallIconRole
	switch s.FileSmallIconRole {
	case "Application (OpenIDEA) Icon", "application":
		s.FileSmallIconRole = "application"
	case "None", "none":
		s.FileSmallIconRole = "none"
	}

	// FileTimeMode
	switch s.FileTimeMode {
	case "Project Time", "Project Total Time", "project":
		s.FileTimeMode = "project"
	case "File Open Time", "file":
		s.FileTimeMode = "file"
	case "None", "none":
		s.FileTimeMode = "none"
	}

	// TimeMode (fallback)
	switch s.TimeMode {
	case "Project Time", "Project Total Time", "project":
		s.TimeMode = "project"
	case "File Open Time", "file":
		s.TimeMode = "file"
	case "None", "none":
		s.TimeMode = "none"
	}

	// ProjectFirstLine
	switch s.ProjectFirstLine {
	case "Project Name (Branch)", "Project Name - VCS branch":
		s.ProjectFirstLine = "{project} ({branch})"
	case "Project Name only":
		s.ProjectFirstLine = "{project}"
	case "Empty":
		s.ProjectFirstLine = ""
	}

	// ProjectSecondLine
	switch s.ProjectSecondLine {
	case "Navigating project":
		s.ProjectSecondLine = "Navigating project"
	case "In project", "Idle in project":
		s.ProjectSecondLine = "Idle in project"
	case "Empty":
		s.ProjectSecondLine = ""
	}

	// ProjectTimeMode
	switch s.ProjectTimeMode {
	case "Project Time", "Project Total Time", "project":
		s.ProjectTimeMode = "project"
	case "None", "none":
		s.ProjectTimeMode = "none"
	}

	// ApplicationTheme & LanguageTheme
	if s.ApplicationTheme != "" {
		s.ApplicationTheme = strings.ToLower(strings.ReplaceAll(s.ApplicationTheme, " ", "_"))
	}
	if s.LanguageTheme != "" {
		s.LanguageTheme = strings.ToLower(strings.ReplaceAll(s.LanguageTheme, " ", "_"))
	}
}

// LoadSettings attempts to load settings from a JSON file, falling back to defaults.
func LoadSettings(path string) Settings {
	s := DefaultSettings()
	if path == "" {
		return s
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}

	_ = json.Unmarshal(data, &s)
	return s
}

// ParsedTimeMode maps string configuration to tracker.TimeMode.
func (s Settings) ParsedTimeMode() tracker.TimeMode {
	switch s.TimeMode {
	case "file":
		return tracker.TimeModeFile
	case "none":
		return tracker.TimeModeNone
	default:
		return tracker.TimeModeProject
	}
}

// ParsedDebounce returns the debouncing duration.
func (s Settings) ParsedDebounce() time.Duration {
	if s.DebounceMs <= 0 {
		return 1500 * time.Millisecond
	}
	return time.Duration(s.DebounceMs) * time.Millisecond
}

// ParsedIdleTimeout returns the idle duration threshold.
func (s Settings) ParsedIdleTimeout() time.Duration {
	if s.IdleTimeoutMins <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(s.IdleTimeoutMins) * time.Minute
}
