package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// exitInstanceLocked is returned when another daemon already holds the lock.
const exitInstanceLocked = 3

var errInstanceLocked = errors.New("daemon instance lock is held by another process")

// acquireInstanceLock takes an exclusive, per-user lock that lives as long
// as the returned file stays open. The OS releases it if the process dies.
func acquireInstanceLock() (*os.File, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("openidea-discord-rpc-%d.lock", os.Getuid()))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := lockExclusive(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}
