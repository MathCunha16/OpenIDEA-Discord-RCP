package template

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"OpenIDEA-Discord-Rcp/internal/ide"
)

// Default template formats.
const (
	DefaultFileDetails = "Editing {file}"
	DefaultFileState   = "{project} ({branch})"

	DefaultProjectDetails = "Navigating project"
	DefaultProjectState   = "{project}"

	DefaultIdleDetails = "Idling"
	DefaultIdleState   = "{project}"
)

// Context encapsulates IDE data available for template token replacement.
type Context struct {
	File         string
	FileRelative string
	Project      string
	Branch       string
	Language     string
	Line         int
	LineCount    int
	IDEName      string
	IDEVersion   string
	IsIdle       bool
	PrivacyMode  bool
}

// Engine formats details and state strings using dynamic template tokens and handles CDN asset resolution.
type Engine struct {
	cdn *CDNResolver

	FileDetails    string
	FileState      string
	ProjectDetails string
	ProjectState   string
	IdleDetails    string
	IdleState      string
}

// NewEngine creates an Engine initialized with default rich presence templates.
func NewEngine(appTheme, langTheme string) *Engine {
	return &Engine{
		cdn:            NewCDNResolver(appTheme, langTheme),
		FileDetails:    DefaultFileDetails,
		FileState:      DefaultFileState,
		ProjectDetails: DefaultProjectDetails,
		ProjectState:   DefaultProjectState,
		IdleDetails:    DefaultIdleDetails,
		IdleState:      DefaultIdleState,
	}
}

// SetThemes updates the underlying CDN icon themes.
func (e *Engine) SetThemes(appTheme, langTheme string) {
	e.cdn.SetThemes(appTheme, langTheme)
}

// SetTheme updates both themes (backward compatibility).
func (e *Engine) SetTheme(theme string) {
	e.cdn.SetTheme(theme)
}

// Theme returns the language theme.
func (e *Engine) Theme() string {
	return e.cdn.Theme()
}

// RenderResult contains the sanitized text fields and CDN URLs for Discord Rich Presence.
type RenderResult struct {
	Details        string
	State          string
	LargeImageURL  string
	LargeImageText string
	SmallImageURL  string
	SmallImageText string
}

// Render compiles presence fields for an EditorEvent with boundary sanitization and privacy filtering.
func (e *Engine) Render(ev *ide.EditorEvent, isIdle bool) RenderResult {
	ideName := "OpenIDEA"
	ideVersion := "1.4.0"

	// Fallback project name
	projectName := ev.Project
	if projectName == "" {
		projectName = "OpenIDEA"
	} else if filepath.IsAbs(projectName) {
		projectName = filepath.Base(projectName)
	}

	branchName := ev.Branch
	if branchName == "" {
		branchName = "main"
	}

	fileRelative := ev.File
	if ev.Project != "" && strings.HasPrefix(ev.File, ev.Project) {
		if rel, err := filepath.Rel(ev.Project, ev.File); err == nil {
			fileRelative = rel
		}
	}

	langInfo := ide.DetectLanguage(ev.File, ev.Language)

	ctx := Context{
		File:         filepath.Base(ev.File),
		FileRelative: fileRelative,
		Project:      projectName,
		Branch:       branchName,
		Language:     langInfo.Name,
		Line:         ev.CursorLine,
		LineCount:    ev.TotalLines,
		IDEName:      ideName,
		IDEVersion:   ideVersion,
		IsIdle:       isIdle,
		PrivacyMode:  ev.PrivacyMode,
	}

	var rawDetails, rawState string
	var largeImageURL, largeImageText string
	var smallImageURL, smallImageText string

	// Small image is always the IDE identity
	smallImageURL = e.cdn.ApplicationIconURL("openidea")
	smallImageText = fmt.Sprintf("%s %s", ideName, ideVersion)

	if isIdle {
		rawDetails = e.formatTemplate(e.IdleDetails, ctx)
		rawState = e.formatTemplate(e.IdleState, ctx)
		largeImageURL = e.cdn.ApplicationIconURL("openidea")
		largeImageText = "Idling"
	} else if ev.File != "" {
		// Active file rendering
		if ev.PrivacyMode {
			rawDetails = ide.PrivacyModeText
		} else if ide.IsSensitiveFile(ev.File) {
			rawDetails = ide.ProtectedConfigText
		} else {
			rawDetails = e.formatTemplate(e.FileDetails, ctx)
		}

		rawState = e.formatTemplate(e.FileState, ctx)

		// Large image represents the active language/file type
		largeImageURL = e.cdn.LanguageIconURL(langInfo.LargeImage)
		largeImageText = langInfo.Name
	} else {
		// Project only (no file currently active)
		rawDetails = e.formatTemplate(e.ProjectDetails, ctx)
		rawState = e.formatTemplate(e.ProjectState, ctx)

		largeImageURL = e.cdn.ApplicationIconURL("openidea")
		largeImageText = projectName
	}

	return RenderResult{
		Details:        FitDiscordString(rawDetails),
		State:          FitDiscordString(rawState),
		LargeImageURL:  largeImageURL,
		LargeImageText: FitDiscordString(largeImageText),
		SmallImageURL:  smallImageURL,
		SmallImageText: FitDiscordString(smallImageText),
	}
}

// formatTemplate replaces dynamic variable tokens in the format string.
func (e *Engine) formatTemplate(template string, ctx Context) string {
	res := template

	res = strings.ReplaceAll(res, "{file}", ctx.File)
	res = strings.ReplaceAll(res, "{file_relative}", ctx.FileRelative)
	res = strings.ReplaceAll(res, "{project}", ctx.Project)
	res = strings.ReplaceAll(res, "{branch}", ctx.Branch)
	res = strings.ReplaceAll(res, "{language}", ctx.Language)
	res = strings.ReplaceAll(res, "{ide_name}", ctx.IDEName)
	res = strings.ReplaceAll(res, "{ide_version}", ctx.IDEVersion)

	if ctx.Line > 0 {
		res = strings.ReplaceAll(res, "{line}", strconv.Itoa(ctx.Line))
	} else {
		res = strings.ReplaceAll(res, "{line}", "")
	}

	if ctx.LineCount > 0 {
		res = strings.ReplaceAll(res, "{line_count}", strconv.Itoa(ctx.LineCount))
	} else {
		res = strings.ReplaceAll(res, "{line_count}", "")
	}

	return strings.TrimSpace(res)
}
