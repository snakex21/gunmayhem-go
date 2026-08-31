//go:build !windows

package updater

import (
	"errors"
	"os/exec"
	"time"
)

func waitForProcessExit(pid int, timeout time.Duration) error {
	return errors.New("automatic updater is currently supported on Windows only")
}

func startHidden(cmd *exec.Cmd) error { return cmd.Start() }

func scheduleSelfDelete(path string) error { return nil }
