package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"OpenIDEA-Discord-Rcp/internal/config"
	"OpenIDEA-Discord-Rcp/internal/discord"
	"OpenIDEA-Discord-Rcp/internal/git"
	"OpenIDEA-Discord-Rcp/internal/ide"
	"OpenIDEA-Discord-Rcp/internal/template"
	"OpenIDEA-Discord-Rcp/internal/tracker"
)

func getSocketPath() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "openidea-discord-rpc.sock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("openidea-discord-rpc-%d.sock", os.Getuid()))
}

func main() {
	topic := flag.String("topic", "", "Event topic (e.g. editor-focused, file-opened)")
	payload := flag.String("payload", "", "Event payload")
	filePath := flag.String("file", "", "Active file path")
	projectPath := flag.String("project", "", "Project root directory")
	branchName := flag.String("branch", "", "Git branch name")
	isShutdown := flag.Bool("shutdown", false, "Shut down the running daemon")
	isReconnect := flag.Bool("reconnect", false, "Trigger Discord IPC reconnection")
	isDaemon := flag.Bool("daemon", false, "Force daemon server mode")
	configPath := flag.String("config", "", "Path to optional settings JSON file")
	flag.Parse()

	sockPath := getSocketPath()

	// If shutdown command is requested
	if *isShutdown {
		conn, err := net.Dial("unix", sockPath)
		if err == nil {
			_ = json.NewEncoder(conn).Encode(ide.EditorEvent{Type: "shutdown"})
			_ = conn.Close()
		}
		os.Exit(0)
	}

	// Client / Forwarder mode
	isClientInvocation := *topic != "" || *payload != "" || *filePath != "" || *projectPath != "" || *isReconnect
	if isClientInvocation && !*isDaemon {
		runClientForwarder(sockPath, *topic, *payload, *filePath, *projectPath, *branchName, *isReconnect, *configPath)
		return
	}

	// Daemon Server mode
	runDaemonServer(sockPath, *configPath)
}

func runClientForwarder(sockPath, topic, payload, filePath, projectPath, branchName string, isReconnect bool, configPath string) {
	// Try connecting to existing daemon
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		// Daemon is not running, spawn it in background
		spawnDaemon(configPath)

		// Wait up to 2 seconds for daemon socket to be ready
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
			conn, err = net.Dial("unix", sockPath)
			if err == nil {
				break
			}
		}
	}

	if err != nil || conn == nil {
		return
	}
	defer conn.Close()

	targetFile := filePath
	if targetFile == "" {
		targetFile = payload
	}

	eventType := topic
	if isReconnect {
		eventType = "reconnect"
	} else if eventType == "" {
		eventType = "editor-focused"
	}

	ev := ide.EditorEvent{
		Type:    eventType,
		File:    targetFile,
		Project: projectPath,
		Branch:  branchName,
	}

	_ = json.NewEncoder(conn).Encode(ev)
}

func spawnDaemon(configPath string) {
	self, err := os.Executable()
	if err != nil {
		self = "openidea-discord-rpc-daemon"
	}

	args := []string{"--daemon"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	cmd := exec.Command(self, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	logPath := filepath.Join(os.TempDir(), "openidea-discord-rpc.log")
	if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	_ = cmd.Start()
}

func runDaemonServer(sockPath string, configPath string) {
	logger := log.New(os.Stderr, "[openidea-discord-rpc] ", log.LstdFlags)
	logger.Println("Starting OpenIDEA Discord Rich Presence sidecar daemon...")

	cfg := config.LoadSettings(configPath)

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

	_ = os.Remove(sockPath)

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		logger.Fatalf("Failed to bind Unix socket at %s: %v", sockPath, err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(sockPath)
	}()

	client := discord.NewClient(cfg.ApplicationID)
	sessionTracker := tracker.NewSessionTracker(cfg.ParsedIdleTimeout(), cfg.ParsedTimeMode())
	engine := template.NewEngine(cfg.Theme)

	eventChan := make(chan *ide.EditorEvent, 32)

	// Accept incoming connections from Unix domain socket
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					logger.Printf("Socket accept error: %v", err)
					return
				}
			}

			go func(c net.Conn) {
				defer c.Close()
				reader := ide.NewEventReader(c)
				for {
					ev, err := reader.ReadNext()
					if err != nil {
						return
					}
					select {
					case eventChan <- ev:
					case <-ctx.Done():
						return
					}
				}
			}(conn)
		}
	}()

	// Read from stdin if stdin is provided via pipe or redirect
	go func() {
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			reader := ide.NewEventReader(os.Stdin)
			for {
				ev, err := reader.ReadNext()
				if err != nil {
					return
				}
				select {
				case eventChan <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Run coordinator loop
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
		} else {
			logger.Printf("Updated Discord Rich Presence: %s | %s", rendered.Details, rendered.State)
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

			if ev.Type == "shutdown" {
				logger.Println("Received shutdown request, terminating...")
				return
			}

			if ev.Type == "reconnect" {
				logger.Println("Reconnecting to Discord IPC...")
				_ = client.Close()
				connectToDiscord()
				if client.IsConnected() && currentEvent != nil {
					publishPresence()
				}
				continue
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
			sessionTracker.OnActivity(ev.File)
			currentEvent = ev

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
			if sessionTracker.CheckIdle(now) {
				logger.Println("User became idle, updating presence...")
				publishPresence()
			}

		case <-reconnectTicker.C:
			if !client.IsConnected() {
				connectToDiscord()
				if client.IsConnected() {
					publishPresence()
				}
			}
		}
	}
}
