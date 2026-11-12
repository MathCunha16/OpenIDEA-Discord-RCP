//go:build windows

package main

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func lockExclusive(f *os.File) error {
	ol := new(windows.Overlapped)
	err := windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, ol,
	)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return errInstanceLocked
	}
	return err
}
