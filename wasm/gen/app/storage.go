package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"OpenIDEA-Discord-Rcp/pkg/config"
)

const (
	StorageKeySessionStart = "discord-rpc.session-start"
	StorageKeyLastSocket   = "discord-rpc.last-socket"
	StorageKeyPaused       = "discord-rpc.paused"
	StorageKeyLastActivity = "discord-rpc.last-activity"
	StorageKeySettings     = "discord-rpc.settings"
)

// StorageManager handles persistent state across plugin activations and reclaims.
type StorageManager struct {
	mem map[string]string
}

func NewStorageManager() *StorageManager {
	return &StorageManager{
		mem: make(map[string]string),
	}
}

// GetString reads a string value from host storage.
func (s *StorageManager) GetString(key string) string {
	if !hostAvailable {
		if s.mem != nil {
			return s.mem[key]
		}
		return ""
	}
	res := hostStorageGet(key)
	if res.IsOk() {
		opt := res.Ok()
		if opt.IsSome() {
			return opt.Some()
		}
	}
	return ""
}

// SetString persists a string value into host storage.
func (s *StorageManager) SetString(key, val string) error {
	if !hostAvailable {
		if s.mem == nil {
			s.mem = make(map[string]string)
		}
		s.mem[key] = val
		return nil
	}
	res := hostStorageSet(key, val)
	if res.IsErr() {
		return fmt.Errorf("storage set %s: %s", key, formatError(res.Err()))
	}
	return nil
}

// Delete removes a key from host storage.
func (s *StorageManager) Delete(key string) error {
	if !hostAvailable {
		if s.mem != nil {
			delete(s.mem, key)
		}
		return nil
	}
	res := hostStorageDelete(key)
	if res.IsErr() {
		return fmt.Errorf("storage delete %s: %s", key, formatError(res.Err()))
	}
	return nil
}

// GetSessionStartWithGap retrieves or resets the active session start timestamp in Unix epoch seconds.
// If elapsed time since lastActivity exceeds gapSeconds (e.g. 30 minutes, or after machine sleep/reboot),
// a new session start timestamp is initialized and persisted.
func (s *StorageManager) GetSessionStartWithGap(gapSeconds int64) int64 {
	now := time.Now().Unix()
	if gapSeconds <= 0 {
		gapSeconds = 30 * 60 // 30 minutes default
	}
	lastActivity := s.GetLastActivity()
	if lastActivity > 0 && (now-lastActivity) >= gapSeconds {
		_ = s.SetString(StorageKeySessionStart, strconv.FormatInt(now, 10))
		return now
	}

	raw := s.GetString(StorageKeySessionStart)
	if raw != "" {
		if val, err := strconv.ParseInt(raw, 10, 64); err == nil && val > 0 {
			return val
		}
	}
	_ = s.SetString(StorageKeySessionStart, strconv.FormatInt(now, 10))
	return now
}

// GetSessionStart retrieves or initializes the active session start timestamp in Unix epoch seconds,
// applying the default 30-minute session gap.
func (s *StorageManager) GetSessionStart() int64 {
	return s.GetSessionStartWithGap(30 * 60)
}

// ResetSessionStart forces a new session start timestamp.
func (s *StorageManager) ResetSessionStart() int64 {
	now := time.Now().Unix()
	_ = s.SetString(StorageKeySessionStart, strconv.FormatInt(now, 10))
	return now
}

// GetLastActivity retrieves the epoch timestamp of the last recorded editor action.
func (s *StorageManager) GetLastActivity() int64 {
	raw := s.GetString(StorageKeyLastActivity)
	if raw != "" {
		if val, err := strconv.ParseInt(raw, 10, 64); err == nil && val > 0 {
			return val
		}
	}
	return time.Now().Unix()
}

// RecordActivity records the current timestamp as the last activity.
func (s *StorageManager) RecordActivity() {
	now := time.Now().Unix()
	_ = s.SetString(StorageKeyLastActivity, strconv.FormatInt(now, 10))
}

// IsPaused returns true if the user toggled presence off.
func (s *StorageManager) IsPaused() bool {
	return s.GetString(StorageKeyPaused) == "true"
}

// SetPaused updates the pause state.
func (s *StorageManager) SetPaused(paused bool) {
	if paused {
		_ = s.SetString(StorageKeyPaused, "true")
	} else {
		_ = s.Delete(StorageKeyPaused)
	}
}

// GetPreferredSocket returns the last working socket path cached in host storage.
func (s *StorageManager) GetPreferredSocket() string {
	return s.GetString(StorageKeyLastSocket)
}

// SavePreferredSocket caches the successfully connected socket path.
func (s *StorageManager) SavePreferredSocket(socketPath string) {
	if socketPath != "" {
		_ = s.SetString(StorageKeyLastSocket, socketPath)
	}
}

// projectSettingsKey returns a namespaced storage key for the given project.
// We use a 16-byte SHA-256 hex suffix so the key length is constant (53 bytes)
// and comfortably within OpenIDEA's 128-byte key limit.
func projectSettingsKey(projectRoot string) string {
	if projectRoot == "" {
		return StorageKeySettings
	}
	sum := sha256.Sum256([]byte(projectRoot))
	return "discord-rpc.settings." + hex.EncodeToString(sum[:16])
}

// LoadSettings returns the settings snapshot persisted for projectRoot,
// and whether a verified snapshot exists. A missing or corrupt snapshot
// returns DefaultSettings and false. We deliberately avoid falling back to
// the legacy global StorageKeySettings key, preventing any contaminated
// v0.3.1 snapshot from leaking into new project instances.
func (s *StorageManager) LoadSettings(projectRoot string) (config.Settings, bool) {
	key := projectSettingsKey(projectRoot)
	raw := s.GetString(key)
	if raw == "" {
		return config.DefaultSettings(), false
	}
	cfg := config.DefaultSettings()
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return config.DefaultSettings(), false
	}
	return cfg, true
}

// HasVerifiedSettings reports whether a persisted snapshot exists for the given project.
func (s *StorageManager) HasVerifiedSettings(projectRoot string) bool {
	key := projectSettingsKey(projectRoot)
	return s.GetString(key) != ""
}

// SaveSettings persists the effective settings so a later activation can
// rebuild them even when the host's settings lookup returns nothing.
// It also purges any legacy un-namespaced snapshot so old state is cleaned up.
func (s *StorageManager) SaveSettings(projectRoot string, cfg config.Settings) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	// Purge legacy global key if it exists
	if s.GetString(StorageKeySettings) != "" {
		_ = s.Delete(StorageKeySettings)
	}
	key := projectSettingsKey(projectRoot)
	return s.SetString(key, string(data))
}
