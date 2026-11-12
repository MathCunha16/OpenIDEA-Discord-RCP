package main

import (
	"errors"
	"os"
	"path/filepath"
)

// exitInstanceLocked is returned when another daemon already holds the lock.
const exitInstanceLocked = 3

var errInstanceLocked = errors.New("daemon instance lock is held by another process")

// daemonStateDir is the per-user directory for the daemon's log and lock.
// Only the current user can access it, so no symlink or hijack is possible.
func daemonStateDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "openidea-discord-rpc")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// acquireInstanceLock takes an exclusive lock that lives as long as the
// returned file stays open. The OS releases it if the process dies.
func acquireInstanceLock() (*os.File, error) {
	dir, err := daemonStateDir()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "daemon.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := lockExclusive(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}
