package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"OpenIDEA-Discord-Rcp/pkg/tracker"
)

// Default settings values.
const (
	DefaultApplicationID  = "1569761142878896160"
	DefaultAppTheme       = "material"
	DefaultLangTheme      = "classic"
	DefaultDebounceMs     = 1500
	DefaultIdleMinutes    = 5
	DefaultSessionGapMins = 30
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
	SessionTimeoutMins    int    `json:"session_timeout_mins"`
	TimeMode              string `json:"time_mode"` // "project", "file", "none"
	ShowGitButton         bool   `json:"show_git_button"`
	ShowIDEButton         bool   `json:"show_ide_button"`
	CustomButtonLabel     string `json:"custom_button_label"`
	CustomButtonURL       string `json:"custom_button_url"`
	PrivacyMode           bool   `json:"privacy_mode"`
	HideSensitiveFiles    bool   `json:"hide_sensitive_files"`

	// Free-form string templates (with placeholders: {file}, {language}, {project}, {branch})
	Details     string `json:"details"`
	State       string `json:"state"`
	IdleDetails string `json:"idle_details"`
	IdleState   string `json:"idle_state"`
	IdleText    string `json:"idle_text"`

	// Legacy layout fields (maintained for backward compatibility)
	FileFirstLine        string `json:"file_first_line"`
	FileSecondLine       string `json:"file_second_line"`
	PrefixReadingEditing bool   `json:"prefix_reading_editing"`
	FileLargeIconRole    string `json:"file_large_icon_role"` // "language", "application"
	FileSmallIconRole    string `json:"file_small_icon_role"` // "application", "none"
	FileTimeMode         string `json:"file_time_mode"`       // "file", "project", "none"

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
		SessionTimeoutMins:    DefaultSessionGapMins,
		TimeMode:              "project",
		ShowGitButton:         true,
		ShowIDEButton:         true,
		CustomButtonLabel:     "",
		CustomButtonURL:       "",
		PrivacyMode:           false,
		HideSensitiveFiles:    true,

		// Modern template defaults
		Details:     "Editing {file}",
		State:       "{project} ({branch})",
		IdleDetails: "Browsing {project}",
		IdleState:   "",
		IdleText:    "Idle in {project}",

		// Legacy field defaults
		FileFirstLine:        "{project} ({branch})",
		FileSecondLine:       "Editing {file}",
		PrefixReadingEditing: true,
		FileLargeIconRole:    "language",
		FileSmallIconRole:    "application",
		FileTimeMode:         "project",
		ProjectFirstLine:     "{project} ({branch})",
		ProjectSecondLine:    "Navigating project",
		ProjectTimeMode:      "project",
	}
}

// UnmarshalJSON supports decoding both camelCase and snake_case configurations.
func (s *Settings) UnmarshalJSON(data []byte) error {
	type RawSettings struct {
		Enabled                    *bool  `json:"enabled"`
		ApplicationID              string `json:"application_id"`
		ApplicationIDCamel         string `json:"applicationId"`
		ApplicationTheme           string `json:"application_theme"`
		ApplicationThemeCamel      string `json:"applicationTheme"`
		LanguageTheme              string `json:"language_theme"`
		LanguageThemeCamel         string `json:"languageTheme"`
		Theme                      string `json:"theme"`
		DebounceMs                 *int   `json:"debounce_ms"`
		DebounceMsCamel            *int   `json:"debounceMs"`
		DetectIdle                 *bool  `json:"detect_idle"`
		DetectIdleCamel            *bool  `json:"detectIdle"`
		IdleTimeoutMins            *int   `json:"idle_timeout_mins"`
		IdleTimeoutMinsCamel       *int   `json:"idleTimeoutMins"`
		WhileIdle                  string `json:"while_idle"`
		WhileIdleCamel             string `json:"whileIdle"`
		ResetOpenTimeOnReturn      *bool  `json:"reset_open_time_on_return"`
		ResetOpenTimeOnReturnCamel *bool  `json:"resetOpenTimeOnReturn"`
		SessionTimeoutMins         *int   `json:"session_timeout_mins"`
		SessionTimeoutMinsCamel    *int   `json:"sessionTimeoutMins"`
		TimeMode                   string `json:"time_mode"`
		TimeModeCamel              string `json:"timeMode"`
		ShowGitButton              *bool  `json:"show_git_button"`
		ShowGitButtonCamel         *bool  `json:"showGitButton"`
		ShowIDEButton              *bool  `json:"show_ide_button"`
		ShowIDEButtonCamel         *bool  `json:"showIDEButton"`
		CustomButtonLabel          string `json:"custom_button_label"`
		CustomButtonLabelCamel     string `json:"customButtonLabel"`
		CustomButtonURL            string `json:"custom_button_url"`
		CustomButtonURLCamel       string `json:"customButtonUrl"`
		PrivacyMode                *bool  `json:"privacy_mode"`
		PrivacyModeCamel           *bool  `json:"privacyMode"`
		HideSensitiveFiles         *bool  `json:"hide_sensitive_files"`
		HideSensitiveFilesCamel    *bool  `json:"hideSensitiveFiles"`

		// Modern string templates
		Details          *string `json:"details"`
		State            *string `json:"state"`
		IdleDetails      *string `json:"idle_details"`
		IdleDetailsCamel *string `json:"idleDetails"`
		IdleState        *string `json:"idle_state"`
		IdleStateCamel   *string `json:"idleState"`
		IdleText         *string `json:"idle_text"`
		IdleTextCamel    *string `json:"idleText"`

		// Legacy file layout
		FileFirstLine             *string `json:"file_first_line"`
		FileFirstLineCamel        *string `json:"fileFirstLine"`
		FileSecondLine            *string `json:"file_second_line"`
		FileSecondLineCamel       *string `json:"fileSecondLine"`
		PrefixReadingEditing      *bool   `json:"prefix_reading_editing"`
		PrefixReadingEditingCamel *bool   `json:"prefixReadingEditing"`
		FileLargeIconRole         string  `json:"file_large_icon_role"`
		FileLargeIconRoleCamel    string  `json:"fileLargeIconRole"`
		FileSmallIconRole         string  `json:"file_small_icon_role"`
		FileSmallIconRoleCamel    string  `json:"fileSmallIconRole"`
		FileTimeMode              string  `json:"file_time_mode"`
		FileTimeModeCamel         string  `json:"fileTimeMode"`

		// Legacy project layout
		ProjectFirstLine       *string `json:"project_first_line"`
		ProjectFirstLineCamel  *string `json:"projectFirstLine"`
		ProjectSecondLine      *string `json:"project_second_line"`
		ProjectSecondLineCamel *string `json:"projectSecondLine"`
		ProjectTimeMode        string  `json:"project_time_mode"`
		ProjectTimeModeCamel   string  `json:"projectTimeMode"`
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

	if raw.SessionTimeoutMins != nil {
		s.SessionTimeoutMins = *raw.SessionTimeoutMins
	} else if raw.SessionTimeoutMinsCamel != nil {
		s.SessionTimeoutMins = *raw.SessionTimeoutMinsCamel
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

	if raw.CustomButtonLabel != "" {
		s.CustomButtonLabel = raw.CustomButtonLabel
	} else if raw.CustomButtonLabelCamel != "" {
		s.CustomButtonLabel = raw.CustomButtonLabelCamel
	}

	if raw.CustomButtonURL != "" {
		s.CustomButtonURL = raw.CustomButtonURL
	} else if raw.CustomButtonURLCamel != "" {
		s.CustomButtonURL = raw.CustomButtonURLCamel
	}

	if raw.PrivacyMode != nil {
		s.PrivacyMode = *raw.PrivacyMode
	} else if raw.PrivacyModeCamel != nil {
		s.PrivacyMode = *raw.PrivacyModeCamel
	}

	if raw.HideSensitiveFiles != nil {
		s.HideSensitiveFiles = *raw.HideSensitiveFiles
	} else if raw.HideSensitiveFilesCamel != nil {
		s.HideSensitiveFiles = *raw.HideSensitiveFilesCamel
	} else {
		s.HideSensitiveFiles = true // Secure default when missing from JSON (e.g. older v0.3.x snapshots)
	}

	hasModern := raw.Details != nil || raw.State != nil ||
		raw.IdleDetails != nil || raw.IdleDetailsCamel != nil ||
		raw.IdleState != nil || raw.IdleStateCamel != nil ||
		raw.IdleText != nil || raw.IdleTextCamel != nil

	if raw.Details != nil {
		s.Details = *raw.Details
	}
	if raw.State != nil {
		s.State = *raw.State
	}
	if raw.IdleDetails != nil {
		s.IdleDetails = *raw.IdleDetails
	} else if raw.IdleDetailsCamel != nil {
		s.IdleDetails = *raw.IdleDetailsCamel
	}
	if raw.IdleState != nil {
		s.IdleState = *raw.IdleState
	} else if raw.IdleStateCamel != nil {
		s.IdleState = *raw.IdleStateCamel
	}
	if raw.IdleText != nil {
		s.IdleText = *raw.IdleText
	} else if raw.IdleTextCamel != nil {
		s.IdleText = *raw.IdleTextCamel
	}

	// Legacy file layout
	hasLegacy := false
	if raw.FileFirstLine != nil {
		s.FileFirstLine = *raw.FileFirstLine
		hasLegacy = true
	} else if raw.FileFirstLineCamel != nil {
		s.FileFirstLine = *raw.FileFirstLineCamel
		hasLegacy = true
	}

	if raw.FileSecondLine != nil {
		s.FileSecondLine = *raw.FileSecondLine
		hasLegacy = true
	} else if raw.FileSecondLineCamel != nil {
		s.FileSecondLine = *raw.FileSecondLineCamel
		hasLegacy = true
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

	// Legacy project layout
	if raw.ProjectFirstLine != nil {
		s.ProjectFirstLine = *raw.ProjectFirstLine
		hasLegacy = true
	} else if raw.ProjectFirstLineCamel != nil {
		s.ProjectFirstLine = *raw.ProjectFirstLineCamel
		hasLegacy = true
	}

	if raw.ProjectSecondLine != nil {
		s.ProjectSecondLine = *raw.ProjectSecondLine
		hasLegacy = true
	} else if raw.ProjectSecondLineCamel != nil {
		s.ProjectSecondLine = *raw.ProjectSecondLineCamel
		hasLegacy = true
	}

	if raw.ProjectTimeMode != "" {
		s.ProjectTimeMode = raw.ProjectTimeMode
	} else if raw.ProjectTimeModeCamel != "" {
		s.ProjectTimeMode = raw.ProjectTimeModeCamel
	}

	s.NormalizePresets()

	// Only migrate from legacy configuration if modern template keys were not provided at all
	if !hasModern && hasLegacy {
		if s.FileSecondLine != "" {
			s.Details = s.FileSecondLine
		}
		if s.FileFirstLine != "" {
			s.State = s.FileFirstLine
		}
		if s.ProjectSecondLine != "" {
			s.IdleDetails = s.ProjectSecondLine
		}
		if s.ProjectFirstLine != "" {
			s.IdleState = s.ProjectFirstLine
		}
	}

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

// ParsedSessionTimeout returns the duration of inactivity after which a new session is started.
func (s Settings) ParsedSessionTimeout() time.Duration {
	if s.SessionTimeoutMins <= 0 {
		return 30 * time.Minute
	}
	return time.Duration(s.SessionTimeoutMins) * time.Minute
}

// SettingsGetter retrieves a JSON-encoded setting string for key.
type SettingsGetter func(key string) (jsonStr string, ok bool)

// LoadFromGetter constructs a Settings struct by querying each key from a getter function.
// It starts from the built-in defaults; use LoadFromGetterWithBase to preserve
// previously persisted values when the host omits a key.
func LoadFromGetter(get SettingsGetter) Settings {
	return LoadFromGetterWithBase(get, DefaultSettings())
}

// LoadFromGetterWithBase overlays host-provided settings on top of base.
// Keys the getter reports as missing (ok == false) keep their value from base,
// so a transient host lookup failure during reactivation cannot silently
// restore defaults (which would, for example, disable Privacy Mode).
func LoadFromGetterWithBase(get SettingsGetter, base Settings) Settings {
	s := base
	if get == nil {
		s.NormalizePresets()
		return s
	}

	decodeBool := func(val string, target *bool) {
		val = strings.TrimSpace(val)
		if val == "true" {
			*target = true
		} else if val == "false" {
			*target = false
		} else {
			var b bool
			if err := json.Unmarshal([]byte(val), &b); err == nil {
				*target = b
			}
		}
	}

	decodeString := func(val string, target *string) {
		val = strings.TrimSpace(val)
		var str string
		if err := json.Unmarshal([]byte(val), &str); err == nil {
			*target = str
		} else if val != "" {
			*target = strings.Trim(val, `"`)
		}
	}

	decodeInt := func(val string, target *int) {
		val = strings.TrimSpace(val)
		var n int
		if err := json.Unmarshal([]byte(val), &n); err == nil {
			*target = n
		} else if parsed, err := strconv.Atoi(val); err == nil {
			*target = parsed
		}
	}

	if val, ok := get("enabled"); ok {
		decodeBool(val, &s.Enabled)
	}
	if val, ok := get("detectIdle"); ok {
		decodeBool(val, &s.DetectIdle)
	}
	if val, ok := get("idleTimeoutMins"); ok {
		decodeInt(val, &s.IdleTimeoutMins)
	}
	if val, ok := get("whileIdle"); ok {
		decodeString(val, &s.WhileIdle)
	}
	if val, ok := get("resetOpenTimeOnReturn"); ok {
		decodeBool(val, &s.ResetOpenTimeOnReturn)
	}
	if val, ok := get("sessionTimeoutMins"); ok {
		decodeInt(val, &s.SessionTimeoutMins)
	}

	// Modern templates
	var hasModernGetter bool
	if val, ok := get("details"); ok {
		decodeString(val, &s.Details)
		hasModernGetter = true
	}
	if val, ok := get("state"); ok {
		decodeString(val, &s.State)
		hasModernGetter = true
	}
	if val, ok := get("idleDetails"); ok {
		decodeString(val, &s.IdleDetails)
		hasModernGetter = true
	}
	if val, ok := get("idleState"); ok {
		decodeString(val, &s.IdleState)
		hasModernGetter = true
	}
	if val, ok := get("idleText"); ok {
		decodeString(val, &s.IdleText)
		hasModernGetter = true
	}

	// Legacy layout keys
	var hasLegacyGetter bool
	if val, ok := get("fileFirstLine"); ok {
		decodeString(val, &s.FileFirstLine)
		hasLegacyGetter = true
	}
	if val, ok := get("fileSecondLine"); ok {
		decodeString(val, &s.FileSecondLine)
		hasLegacyGetter = true
	}
	if val, ok := get("projectFirstLine"); ok {
		decodeString(val, &s.ProjectFirstLine)
		hasLegacyGetter = true
	}
	if val, ok := get("projectSecondLine"); ok {
		decodeString(val, &s.ProjectSecondLine)
		hasLegacyGetter = true
	}
	if val, ok := get("prefixReadingEditing"); ok {
		decodeBool(val, &s.PrefixReadingEditing)
	}

	if val, ok := get("timeMode"); ok {
		decodeString(val, &s.TimeMode)
	}
	if val, ok := get("applicationTheme"); ok {
		decodeString(val, &s.ApplicationTheme)
	}
	if val, ok := get("languageTheme"); ok {
		decodeString(val, &s.LanguageTheme)
	}
	if val, ok := get("privacyMode"); ok {
		decodeBool(val, &s.PrivacyMode)
	}
	if val, ok := get("hideSensitiveFiles"); ok {
		decodeBool(val, &s.HideSensitiveFiles)
	}
	if val, ok := get("showGitButton"); ok {
		decodeBool(val, &s.ShowGitButton)
	}
	if val, ok := get("showIDEButton"); ok {
		decodeBool(val, &s.ShowIDEButton)
	}
	if val, ok := get("customButtonLabel"); ok {
		decodeString(val, &s.CustomButtonLabel)
	}
	if val, ok := get("customButtonUrl"); ok {
		decodeString(val, &s.CustomButtonURL)
	}

	s.NormalizePresets()

	// Only migrate from legacy configuration if modern template keys were not supplied
	if !hasModernGetter && hasLegacyGetter {
		if s.FileSecondLine != "" {
			s.Details = s.FileSecondLine
		}
		if s.FileFirstLine != "" {
			s.State = s.FileFirstLine
		}
		if s.ProjectSecondLine != "" {
			s.IdleDetails = s.ProjectSecondLine
		}
		if s.ProjectFirstLine != "" {
			s.IdleState = s.ProjectFirstLine
		}
	}

	return s
}
