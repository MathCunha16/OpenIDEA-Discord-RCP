//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// On Windows, DETACHED_PROCESS (0x00000008) runs the process detached from the current console.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008,
	}
}
