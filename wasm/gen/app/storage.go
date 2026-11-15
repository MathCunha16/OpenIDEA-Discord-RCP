package app

import (
	"fmt"
	"strconv"
	"time"

	"wit_component/openidea_plugin_storage"
)

const (
	StorageKeySessionStart = "discord-rpc.session-start"
	StorageKeyLastSocket   = "discord-rpc.last-socket"
	StorageKeyPaused       = "discord-rpc.paused"
	StorageKeyLastActivity = "discord-rpc.last-activity"
)

// StorageManager handles persistent state across plugin activations and reclaims.
type StorageManager struct{}

func NewStorageManager() *StorageManager {
	return &StorageManager{}
}

// GetString reads a string value from host storage.
func (s *StorageManager) GetString(key string) string {
	res := openidea_plugin_storage.Get(key)
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
	res := openidea_plugin_storage.Set(key, val)
	if res.IsErr() {
		return fmt.Errorf("storage set %s: %s", key, formatError(res.Err()))
	}
	return nil
}

// Delete removes a key from host storage.
func (s *StorageManager) Delete(key string) error {
	res := openidea_plugin_storage.Delete(key)
	if res.IsErr() {
		return fmt.Errorf("storage delete %s: %s", key, formatError(res.Err()))
	}
	return nil
}

// GetSessionStart retrieves or initializes the active session start timestamp in Unix epoch seconds.
func (s *StorageManager) GetSessionStart() int64 {
	raw := s.GetString(StorageKeySessionStart)
	if raw != "" {
		if val, err := strconv.ParseInt(raw, 10, 64); err == nil && val > 0 {
			return val
		}
	}
	now := time.Now().Unix()
	_ = s.SetString(StorageKeySessionStart, strconv.FormatInt(now, 10))
	return now
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
