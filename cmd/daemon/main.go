package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"OpenIDEA-Discord-Rcp/internal/config"
	"OpenIDEA-Discord-Rcp/internal/discord"
	"OpenIDEA-Discord-Rcp/internal/git"
	"OpenIDEA-Discord-Rcp/internal/ide"
	"OpenIDEA-Discord-Rcp/internal/template"
	"OpenIDEA-Discord-Rcp/internal/tracker"
)

func main() {
	configPath := flag.String("config", "", "Path to optional settings JSON file")
	flag.Parse()

	// Setup logging to stderr to keep stdout clean for IPC if needed
	logger := log.New(os.Stderr, "[openidea-discord-rpc] ", log.LstdFlags)
	logger.Println("Starting OpenIDEA Discord Rich Presence sidecar daemon...")

	cfg := config.LoadSettings(*configPath)

	// Context for graceful shutdown on SIGINT / SIGTERM or stdin EOF
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

	// Initialize components
	client := discord.NewClient(cfg.ApplicationID)
	sessionTracker := tracker.NewSessionTracker(cfg.ParsedIdleTimeout(), cfg.ParsedTimeMode())
	engine := template.NewEngine(cfg.Theme)

	eventChan := make(chan *ide.EditorEvent, 16)

	// Goroutine 1: Read events from os.Stdin
	go func() {
		reader := ide.NewEventReader(os.Stdin)
		for {
			ev, err := reader.ReadNext()
			if err != nil {
				if err == io.EOF {
					logger.Println("OpenIDEA closed stdin (host shutdown), terminating...")
				} else {
					logger.Printf("Error reading event: %v", err)
				}
				cancel()
				return
			}
			select {
			case eventChan <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Run main coordinator loop (connects, debounces, tracks AFK, updates presence)
	runCoordinator(ctx, client, sessionTracker, engine, cfg, eventChan, logger)

	// Cleanup on shutdown
	logger.Println("Disconnecting from Discord...")
	_ = client.ClearActivity()
	_ = client.Close()
	logger.Println("Daemon gracefully exited.")
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

	reconnectTicker := time.NewTicker(5 * time.Second)
	defer reconnectTicker.Stop()

	var debounceTimer *time.Timer
	var debounceChan <-chan time.Time

	var currentEvent *ide.EditorEvent
	var gitRepo *git.RepositoryInfo

	connectToDiscord := func() {
		if !client.IsConnected() {
			dialCtx, dialCancel := context.WithTimeout(ctx, 2*time.Second)
			err := client.Connect(dialCtx)
			dialCancel()
			if err != nil {
				logger.Printf("Discord connection attempt: %v", err)
			} else {
				logger.Println("Successfully connected to Discord IPC!")
			}
		}
	}

	// Initial connection attempt
	connectToDiscord()

	publishPresence := func() {
		if !client.IsConnected() {
			return
		}

		if currentEvent == nil {
			return
		}

		isIdle := sessionTracker.IsIdle()
		rendered := engine.Render(currentEvent, isIdle)

		// Timestamps
		var timestamps *discord.Timestamps
		startTime := sessionTracker.TimestampStart()
		if startTime > 0 && !isIdle {
			timestamps = &discord.Timestamps{
				Start: startTime,
			}
		}

		// Assets
		assets := &discord.Assets{
			LargeImage: rendered.LargeImageURL,
			LargeText:  rendered.LargeImageText,
			SmallImage: rendered.SmallImageURL,
			SmallText:  rendered.SmallImageText,
		}

		// Buttons
		var buttons []discord.Button
		if cfg.ShowGitButton && gitRepo != nil && gitRepo.WebURL != "" {
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

		if err := client.SetActivity(activity); err != nil {
			logger.Printf("Failed to update activity: %v", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return

		case ev := <-eventChan:
			if ev == nil {
				continue
			}

			// Handle shutdown event explicitly if sent
			if ev.Type == "shutdown" {
				return
			}

			// Handle activity and AFK recovery
			sessionTracker.OnActivity(ev.File)
			currentEvent = ev

			// Update Git repository metadata when project changes
			if ev.Project != "" && (gitRepo == nil || gitRepo.Branch != ev.Branch) {
				repoInfo, err := git.ParseRepository(ev.Project)
				if err == nil && repoInfo != nil {
					gitRepo = repoInfo
				}
			}

			// Reset debounce timer
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.NewTimer(debounceDuration)
			debounceChan = debounceTimer.C

		case <-debounceChan:
			debounceTimer = nil
			debounceChan = nil
			publishPresence()

		case now := <-idleTicker.C:
			// Check if state transitioned to idle
			if sessionTracker.CheckIdle(now) {
				logger.Println("User became idle, updating presence...")
				publishPresence()
			}

		case <-reconnectTicker.C:
			// Reconnect loop with exponential backoff behavior
			if !client.IsConnected() {
				connectToDiscord()
				if client.IsConnected() {
					publishPresence()
				}
			}
		}
	}
}
