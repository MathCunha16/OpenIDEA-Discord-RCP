package template

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"OpenIDEA-Discord-Rcp/pkg/ide"
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
	Sensitive    bool
}

// Engine formats details and state strings using dynamic template tokens and handles CDN asset resolution.
type Engine struct {
	cdn *CDNResolver

	Details     string
	State       string
	IdleDetails string
	IdleState   string
	IdleText    string

	// Legacy aliases
	FileDetails          string
	FileState            string
	ProjectDetails       string
	ProjectState         string
	PrefixReadingEditing bool
}

// NewEngine creates an Engine initialized with default rich presence templates.
func NewEngine(appTheme, langTheme string) *Engine {
	return &Engine{
		cdn:                  NewCDNResolver(appTheme, langTheme),
		Details:              DefaultFileDetails,
		State:                DefaultFileState,
		IdleDetails:          DefaultIdleDetails,
		IdleState:            DefaultIdleState,
		IdleText:             DefaultIdleDetails,
		FileDetails:          DefaultFileDetails,
		FileState:            DefaultFileState,
		ProjectDetails:       DefaultProjectDetails,
		ProjectState:         DefaultProjectState,
		PrefixReadingEditing: true,
	}
}

// CDN returns the underlying CDNResolver.
func (e *Engine) CDN() *CDNResolver {
	return e.cdn
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

	fileRelative := ev.File
	if ev.Project != "" && strings.HasPrefix(ev.File, ev.Project) {
		if rel, err := filepath.Rel(ev.Project, ev.File); err == nil {
			fileRelative = rel
		}
	}

	langInfo := ide.DetectLanguage(ev.File, ev.Language)

	ctx := Context{
		File:         ide.DisambiguateFileName(ev.File, ev.OpenFiles),
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
		Sensitive:    ev.Sensitive,
	}

	var rawDetails, rawState string
	var largeImageURL, largeImageText string
	var smallImageURL, smallImageText string

	// Small image is always the IDE identity
	smallImageURL = e.cdn.ApplicationIconURL("openidea")
	smallImageText = fmt.Sprintf("%s %s", ideName, ideVersion)

	if isIdle {
		if ev.PrivacyMode {
			rawDetails = "Idling"
			rawState = ""
		} else {
			idleTpl := e.IdleText
			if idleTpl == "" {
				idleTpl = e.IdleDetails
			}
			if idleTpl == "" {
				idleTpl = DefaultIdleDetails
			}
			rawDetails = e.formatTemplate(idleTpl, ctx)
			stateTpl := e.IdleState
			rawState = e.formatTemplate(stateTpl, ctx)
		}
		largeImageURL = e.cdn.ApplicationIconURL("openidea")
		largeImageText = "Idling"
	} else if ev.File != "" {
		// Active file rendering
		if ev.PrivacyMode {
			rawDetails = ide.SanitizeFileDetails(ev.File, true)
			rawState = ""
			largeImageURL = e.cdn.LanguageIconURL(langInfo.LargeImage)
			largeImageText = langInfo.Name
		} else if ctx.Sensitive {
			rawDetails = ide.ProtectedConfigText
			stateTpl := e.State
			if stateTpl == "" {
				stateTpl = e.FileState
			}
			sensitiveCtx := ctx
			sensitiveCtx.File = ""
			sensitiveCtx.FileRelative = ""
			rawState = e.formatTemplate(stateTpl, sensitiveCtx)
			largeImageURL = e.cdn.ApplicationIconURL("openidea")
			largeImageText = ideName
		} else {
			detailsTpl := e.Details
			if detailsTpl == "" {
				detailsTpl = e.FileDetails
			}
			rawDetails = e.formatTemplate(detailsTpl, ctx)
			if !e.PrefixReadingEditing && strings.HasPrefix(rawDetails, "Editing ") {
				rawDetails = strings.TrimPrefix(rawDetails, "Editing ")
			}
			stateTpl := e.State
			if stateTpl == "" {
				stateTpl = e.FileState
			}
			rawState = e.formatTemplate(stateTpl, ctx)

			largeImageURL = e.cdn.LanguageIconURL(langInfo.LargeImage)
			largeImageText = langInfo.Name
		}
	} else {
		// Project only (no file currently active)
		if ev.PrivacyMode {
			rawDetails = "In workspace"
			rawState = ""
			largeImageText = ideName
		} else {
			detailsTpl := e.ProjectDetails
			if detailsTpl == "" {
				detailsTpl = e.IdleDetails
			}
			rawDetails = e.formatTemplate(detailsTpl, ctx)
			stateTpl := e.ProjectState
			if stateTpl == "" {
				stateTpl = e.IdleState
			}
			rawState = e.formatTemplate(stateTpl, ctx)
			largeImageText = projectName
		}

		largeImageURL = e.cdn.ApplicationIconURL("openidea")
	}

	finalDetails := ""
	if rawDetails != "" {
		finalDetails = FitDiscordString(rawDetails)
	}

	finalState := ""
	if rawState != "" {
		finalState = FitDiscordString(rawState)
	}

	return RenderResult{
		Details:        finalDetails,
		State:          finalState,
		LargeImageURL:  largeImageURL,
		LargeImageText: FitDiscordString(largeImageText),
		SmallImageURL:  smallImageURL,
		SmallImageText: FitDiscordString(smallImageText),
	}
}

// formatTemplate replaces dynamic variable tokens in the format string and performs sanitization.
func (e *Engine) formatTemplate(template string, ctx Context) string {
	res := template

	res = strings.ReplaceAll(res, "{file}", ctx.File)
	res = strings.ReplaceAll(res, "{file_relative}", ctx.FileRelative)
	res = strings.ReplaceAll(res, "{fileRelative}", ctx.FileRelative)
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
		res = strings.ReplaceAll(res, "{lineCount}", strconv.Itoa(ctx.LineCount))
	} else {
		res = strings.ReplaceAll(res, "{line_count}", "")
		res = strings.ReplaceAll(res, "{lineCount}", "")
	}

	// Smart cleanup of empty brackets/parentheses caused by empty variables (e.g. empty branch)
	res = strings.ReplaceAll(res, "()", "")
	res = strings.ReplaceAll(res, "[]", "")

	// Normalize whitespace
	fields := strings.Fields(res)
	res = strings.Join(fields, " ")

	// Trim leading and trailing separators
	res = strings.Trim(res, " ·-|/:,")
	res = strings.TrimSpace(res)

	if res == "" {
		return ""
	}

	// Truncate to maximum 128 characters
	runes := []rune(res)
	if len(runes) > 128 {
		runes = runes[:128]
	}

	// Discord requires at least 2 characters for activity string fields
	for len(runes) < 2 {
		runes = append(runes, ' ')
	}

	return string(runes)
}
