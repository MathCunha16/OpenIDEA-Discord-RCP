package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"OpenIDEA-Discord-Rcp/internal/config"
	"OpenIDEA-Discord-Rcp/internal/discord"
	"OpenIDEA-Discord-Rcp/internal/git"
	"OpenIDEA-Discord-Rcp/internal/ide"
	"OpenIDEA-Discord-Rcp/internal/template"
	"OpenIDEA-Discord-Rcp/internal/tracker"
)

// DaemonVersion holds the current semver release of the companion daemon,
// typically injected at build time via -ldflags "-X main.DaemonVersion=x.y.z".
var DaemonVersion = "dev"

const maxEventLine = 1 << 20

func main() {
	showVersion := flag.Bool("version", false, "Print daemon version and exit")
	flag.Parse()

	if *showVersion || (len(os.Args) > 1 && os.Args[1] == "version") {
		fmt.Println(DaemonVersion)
		return
	}

	runDaemon()
}

// runDaemon serves one plugin instance: it reads newline-delimited events from
// stdin and exits on EOF, so closing the plugin's stdin stops the daemon.
func runDaemon() {
	lock, err := acquireInstanceLock()
	if err != nil {
		if errors.Is(err, errInstanceLocked) {
			// The running instance owns the log file; do not truncate it.
			os.Exit(exitInstanceLocked)
		}
		log.Fatalf("Failed to acquire instance lock: %v", err)
	}
	defer lock.Close()

	logger := newLogger()
	logger.Println("Starting OpenIDEA Discord Rich Presence sidecar daemon...")

	cfg := config.DefaultSettings()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigChan:
			logger.Printf("Received signal %s, shutting down...", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	client := discord.NewClient(cfg.ApplicationID)
	sessionTracker := tracker.NewSessionTracker(cfg.ParsedIdleTimeout(), cfg.ParsedTimeMode())
	engine := template.NewEngine(cfg.ApplicationTheme, cfg.LanguageTheme)

	eventChan := make(chan *ide.EditorEvent, 32)
	go readEvents(ctx, os.Stdin, eventChan, logger)

	runCoordinator(ctx, client, sessionTracker, engine, cfg, eventChan, logger)

	logger.Println("Disconnecting from Discord...")
	_ = client.ClearActivity()
	_ = client.Close()
	logger.Println("Daemon gracefully exited.")
}

// readEvents decodes one JSON event per line. Malformed lines are skipped; the
// channel is closed when stdin reaches EOF.
func readEvents(ctx context.Context, r io.Reader, out chan<- *ide.EditorEvent, logger *log.Logger) {
	defer close(out)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxEventLine)
	for scanner.Scan() {
		var ev ide.EditorEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			logger.Printf("Ignoring malformed event: %v", err)
			continue
		}
		select {
		case out <- &ev:
		case <-ctx.Done():
			return
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Printf("Event stream error: %v", err)
	}
}

func newLogger() *log.Logger {
	logPath := filepath.Join(os.TempDir(), "openidea-discord-rpc.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return log.New(io.Discard, "", 0)
	}
	return log.New(f, "[openidea-discord-rpc] ", log.LstdFlags)
}

func isSettingsOrConfigFile(path string) bool {
	clean := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(clean)
	if base == "settings.json" || base == "settings.toml" || base == "settings" {
		return true
	}
	if strings.Contains(clean, ".openidea/settings") || strings.Contains(clean, "/.openidea/") {
		return true
	}
	return false
}

func isValidCodeFile(f string) bool {
	clean := strings.TrimSpace(f)
	if clean == "" || strings.HasPrefix(clean, "{") || strings.HasSuffix(clean, "}") || strings.Contains(clean, "\n") {
		return false
	}
	if isSettingsOrConfigFile(clean) {
		return false
	}
	return true
}

func runCoordinator(
	ctx context.Context,
	client discord.Client,
	sessionTracker *tracker.SessionTracker,
	engine *template.Engine,
	cfg config.Settings,
	eventChan <-chan *ide.EditorEvent,
	logger *log.Logger,
) {
	debounceDuration := cfg.ParsedDebounce()
	idleTicker := time.NewTicker(15 * time.Second)
	defer idleTicker.Stop()

	var debounceTimer *time.Timer
	var debounceChan <-chan time.Time

	const minPublishInterval = 2 * time.Second
	var lastPublishTime time.Time
	var throttleTimer *time.Timer
	var throttleChan <-chan time.Time
	var pendingPublish bool

	var settingsDebounceTimer *time.Timer
	var settingsDebounceChan <-chan time.Time

	var currentEvent *ide.EditorEvent
	var gitRepo *git.RepositoryInfo
	var lastPublishedActivity *discord.Activity
	var lastConnectAttempt time.Time
	var loggedOffline bool

	connectToDiscord := func() bool {
		if !cfg.Enabled {
			return false
		}
		if client.IsConnected() {
			return true
		}

		// Don't hammer socket dial faster than once every 5 seconds per attempt
		if time.Since(lastConnectAttempt) < 5*time.Second {
			return false
		}
		lastConnectAttempt = time.Now()

		dialCtx, dialCancel := context.WithTimeout(ctx, 2*time.Second)
		err := client.Connect(dialCtx)
		dialCancel()
		if err != nil {
			if !loggedOffline {
				logger.Printf("Discord IPC not available (is Discord running?): %v", err)
				loggedOffline = true
			}
			return false
		}

		loggedOffline = false
		lastPublishedActivity = nil
		logger.Println("Successfully connected to Discord IPC!")
		return true
	}

	// Apply initial template settings from config
	engine.SetThemes(cfg.ApplicationTheme, cfg.LanguageTheme)
	if cfg.FileFirstLine != "" {
		engine.FileState = cfg.FileFirstLine
	}
	if cfg.FileSecondLine != "" {
		engine.FileDetails = cfg.FileSecondLine
	}
	if cfg.ProjectFirstLine != "" {
		engine.ProjectState = cfg.ProjectFirstLine
	}
	if cfg.ProjectSecondLine != "" {
		engine.ProjectDetails = cfg.ProjectSecondLine
	}
	engine.PrefixReadingEditing = cfg.PrefixReadingEditing

	// Initial connection attempt (lazy)
	connectToDiscord()

	publishPresence := func() {
		if !cfg.Enabled {
			if client.IsConnected() {
				_ = client.ClearActivity()
			}
			lastPublishedActivity = nil
			return
		}

		if !client.IsConnected() {
			if !connectToDiscord() {
				return
			}
		}

		if currentEvent == nil {
			return
		}

		isIdle := cfg.DetectIdle && sessionTracker.IsIdle()
		if isIdle && cfg.WhileIdle == "hide_presence" {
			_ = client.ClearActivity()
			lastPublishedActivity = nil
			return
		}

		rendered := engine.Render(currentEvent, isIdle && cfg.WhileIdle == "show_idling")

		// Timestamps
		var timestamps *discord.Timestamps
		startTime := sessionTracker.TimestampStart()
		if startTime > 0 && !isIdle {
			timestamps = &discord.Timestamps{
				Start: startTime,
			}
		}

		// Assets
		largeImgURL := rendered.LargeImageURL
		largeImgText := rendered.LargeImageText
		smallImgURL := rendered.SmallImageURL
		smallImgText := rendered.SmallImageText

		// Apply role configuration for file view
		if currentEvent.File != "" && !isIdle {
			if cfg.FileLargeIconRole == "application" {
				largeImgURL = rendered.SmallImageURL
				largeImgText = rendered.SmallImageText
				smallImgURL = rendered.LargeImageURL
				smallImgText = rendered.LargeImageText
			}
			if cfg.FileSmallIconRole == "none" {
				smallImgURL = ""
				smallImgText = ""
			}
		}

		assets := &discord.Assets{
			LargeImage: largeImgURL,
			LargeText:  largeImgText,
			SmallImage: smallImgURL,
			SmallText:  smallImgText,
		}

		// Buttons
		var buttons []discord.Button
		if cfg.ShowGitButton && !cfg.PrivacyMode && gitRepo != nil && gitRepo.WebURL != "" {
			buttons = append(buttons, discord.Button{
				Label: "View on GitHub",
				URL:   gitRepo.WebURL,
			})
		}

		if cfg.ShowIDEButton {
			buttons = append(buttons, discord.Button{
				Label: "Get OpenIDEA",
				URL:   "https://github.com/l0ng-ai/openidea",
			})
		}

		activity := &discord.Activity{
			Details:    rendered.Details,
			State:      rendered.State,
			Timestamps: timestamps,
			Assets:     assets,
			Buttons:    buttons,
		}

		if lastPublishedActivity != nil && activity.Equal(lastPublishedActivity) {
			return
		}

		if err := client.SetActivity(activity); err != nil {
			logger.Printf("Failed to update activity: %v", err)
			lastPublishedActivity = nil
		} else {
			lastPublishedActivity = activity
			logger.Printf("Updated Discord Rich Presence: %s | %s", rendered.Details, rendered.State)
		}
	}

	requestPublish := func() {
		if !cfg.Enabled {
			if client.IsConnected() {
				_ = client.ClearActivity()
			}
			lastPublishedActivity = nil
			return
		}

		if throttleTimer != nil {
			pendingPublish = true
			return
		}

		elapsed := time.Since(lastPublishTime)
		if elapsed >= minPublishInterval {
			lastPublishTime = time.Now()
			publishPresence()
		} else {
			pendingPublish = true
			waitDuration := minPublishInterval - elapsed
			throttleTimer = time.NewTimer(waitDuration)
			throttleChan = throttleTimer.C
		}
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-throttleChan:
			throttleTimer = nil
			throttleChan = nil
			if pendingPublish {
				pendingPublish = false
				lastPublishTime = time.Now()
				publishPresence()
			}

		case <-settingsDebounceChan:
			settingsDebounceTimer = nil
			settingsDebounceChan = nil
			requestPublish()

		case ev, ok := <-eventChan:
			if !ok {
				logger.Println("Event stream closed, terminating...")
				return
			}

			if ev.Type == "settings-changed" || ev.Type == "plugin-activated" || ev.Type == "scheduled" {
				if ev.Payload != "" {
					var newCfg config.Settings
					if err := json.Unmarshal([]byte(ev.Payload), &newCfg); err == nil {
						if newCfg.ApplicationID != "" && newCfg.ApplicationID != cfg.ApplicationID {
							logger.Printf("Application ID changed from %s to %s, reconnecting...", cfg.ApplicationID, newCfg.ApplicationID)
							cfg.ApplicationID = newCfg.ApplicationID
							_ = client.Close()
							client = discord.NewClient(cfg.ApplicationID)
							connectToDiscord()
						}

						cfg.Enabled = newCfg.Enabled
						cfg.DetectIdle = newCfg.DetectIdle
						cfg.WhileIdle = newCfg.WhileIdle
						cfg.ResetOpenTimeOnReturn = newCfg.ResetOpenTimeOnReturn

						if newCfg.ApplicationTheme != "" || newCfg.LanguageTheme != "" {
							if newCfg.ApplicationTheme != "" {
								cfg.ApplicationTheme = newCfg.ApplicationTheme
							}
							if newCfg.LanguageTheme != "" {
								cfg.LanguageTheme = newCfg.LanguageTheme
							}
							engine.SetThemes(cfg.ApplicationTheme, cfg.LanguageTheme)
						}

						// Custom templates
						if newCfg.FileFirstLine != "" {
							cfg.FileFirstLine = newCfg.FileFirstLine
							engine.FileState = cfg.FileFirstLine
						}
						if newCfg.FileSecondLine != "" {
							cfg.FileSecondLine = newCfg.FileSecondLine
							engine.FileDetails = cfg.FileSecondLine
						}
						if newCfg.ProjectFirstLine != "" {
							cfg.ProjectFirstLine = newCfg.ProjectFirstLine
							engine.ProjectState = cfg.ProjectFirstLine
						}
						if newCfg.ProjectSecondLine != "" {
							cfg.ProjectSecondLine = newCfg.ProjectSecondLine
							engine.ProjectDetails = cfg.ProjectSecondLine
						}

						cfg.FileLargeIconRole = newCfg.FileLargeIconRole
						cfg.FileSmallIconRole = newCfg.FileSmallIconRole
						cfg.PrefixReadingEditing = newCfg.PrefixReadingEditing
						engine.PrefixReadingEditing = cfg.PrefixReadingEditing

						// Time modes
						if newCfg.TimeMode != "" {
							cfg.TimeMode = newCfg.TimeMode
						} else if newCfg.FileTimeMode != "" {
							cfg.TimeMode = newCfg.FileTimeMode
						}
						if newCfg.FileTimeMode != "" {
							cfg.FileTimeMode = newCfg.FileTimeMode
						}
						if newCfg.ProjectTimeMode != "" {
							cfg.ProjectTimeMode = newCfg.ProjectTimeMode
						}
						sessionTracker.SetTimeMode(cfg.ParsedTimeMode())

						if newCfg.IdleTimeoutMins > 0 {
							cfg.IdleTimeoutMins = newCfg.IdleTimeoutMins
							sessionTracker.SetIdleTimeout(cfg.ParsedIdleTimeout())
						}

						if newCfg.DebounceMs > 0 {
							cfg.DebounceMs = newCfg.DebounceMs
							debounceDuration = cfg.ParsedDebounce()
						}

						cfg.ShowGitButton = newCfg.ShowGitButton
						cfg.ShowIDEButton = newCfg.ShowIDEButton
						cfg.PrivacyMode = newCfg.PrivacyMode

						if ev.Type != "scheduled" {
							logger.Printf("Settings applied live: enabled=%v, appTheme=%s, langTheme=%s, timeMode=%s, privacy=%v, detectIdle=%v",
								cfg.Enabled, cfg.ApplicationTheme, cfg.LanguageTheme, cfg.TimeMode, cfg.PrivacyMode, cfg.DetectIdle)
						}

						if !cfg.Enabled {
							if client.IsConnected() {
								_ = client.ClearActivity()
							}
							lastPublishedActivity = nil
						} else {
							if isValidCodeFile(ev.File) {
								if currentEvent == nil {
									currentEvent = &ide.EditorEvent{Type: "editor-focused"}
								}
								currentEvent.File = ev.File
							}
							if ev.Project != "" {
								if currentEvent == nil {
									currentEvent = &ide.EditorEvent{Type: "editor-focused"}
								}
								currentEvent.Project = ev.Project
							}
							if currentEvent == nil {
								currentEvent = &ide.EditorEvent{
									Type:    "editor-focused",
									File:    ev.File,
									Project: ev.Project,
								}
							}
							if currentEvent != nil {
								if currentEvent.Project == "" && currentEvent.File != "" {
									if gitRoot := git.FindGitRoot(currentEvent.File); gitRoot != "" {
										currentEvent.Project = gitRoot
									}
								}
								if currentEvent.Project != "" && (gitRepo == nil || currentEvent.Branch == "" || gitRepo.Branch != currentEvent.Branch) {
									if repoInfo, err := git.ParseRepository(currentEvent.Project); err == nil && repoInfo != nil {
										gitRepo = repoInfo
										if currentEvent.Branch == "" {
											currentEvent.Branch = repoInfo.Branch
										}
									}
								}
								currentEvent.PrivacyMode = cfg.PrivacyMode
							}
							if !client.IsConnected() {
								connectToDiscord()
							}

							if ev.Type == "plugin-activated" {
								requestPublish()
							} else if ev.Type == "scheduled" {
								requestPublish()
							} else {
								// Debounce rapid settings/theme updates by 350ms
								if settingsDebounceTimer != nil {
									settingsDebounceTimer.Stop()
								}
								settingsDebounceTimer = time.NewTimer(350 * time.Millisecond)
								settingsDebounceChan = settingsDebounceTimer.C
							}
						}
					} else {
						logger.Printf("Failed to unmarshal settings payload: %v", err)
					}
				}
				continue
			}

			// Validate and sanitize active file path
			if !isValidCodeFile(ev.File) {
				if currentEvent != nil && currentEvent.File != "" {
					ev.File = currentEvent.File
				} else {
					ev.File = ""
				}
			}

			// Apply configured global privacy mode
			if cfg.PrivacyMode {
				ev.PrivacyMode = true
			}

			// If project is not set, try to derive from file's git root
			if ev.Project == "" && ev.File != "" {
				if gitRoot := git.FindGitRoot(ev.File); gitRoot != "" {
					ev.Project = gitRoot
				}
			}

			// Parse Git repo metadata
			if ev.Project != "" && (gitRepo == nil || ev.Branch == "" || gitRepo.Branch != ev.Branch) {
				repoInfo, err := git.ParseRepository(ev.Project)
				if err == nil && repoInfo != nil {
					gitRepo = repoInfo
					if ev.Branch == "" {
						ev.Branch = repoInfo.Branch
					}
				}
			}

			// AFK recovery and activity registration
			wasIdle := sessionTracker.OnActivity(ev.File)
			if wasIdle && cfg.ResetOpenTimeOnReturn {
				sessionTracker.ResetProject()
			}

			firstEvent := (currentEvent == nil)
			currentEvent = ev

			// Deliver first event instantly via requestPublish; debounce subsequent rapid transitions
			if firstEvent {
				requestPublish()
			} else {
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.NewTimer(debounceDuration)
				debounceChan = debounceTimer.C
			}

		case <-debounceChan:
			debounceTimer = nil
			debounceChan = nil
			requestPublish()

		case now := <-idleTicker.C:
			if sessionTracker.CheckIdle(now) {
				logger.Println("User became idle, updating presence...")
				requestPublish()
			} else if currentEvent != nil && !client.IsConnected() && cfg.Enabled {
				// Retry connection lazily during periodic idle heartbeat if disconnected
				requestPublish()
			}
		}
	}
}
