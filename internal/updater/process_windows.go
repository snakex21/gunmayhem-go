//go:build windows

package updater

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func waitForProcessExit(pid int, timeout time.Duration) error {
	if pid <= 0 {
		return nil
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		// If the process already vanished between launch and helper startup,
		// there is nothing left to wait for.
		if err == windows.ERROR_INVALID_PARAMETER {
			return nil
		}
		return err
	}
	defer windows.CloseHandle(h)
	ms := uint32(timeout / time.Millisecond)
	result, err := windows.WaitForSingleObject(h, ms)
	if err != nil {
		return err
	}
	if result != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("timed out waiting for PID %d", pid)
	}
	return nil
}

func startHidden(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

func scheduleSelfDelete(path string) error {
	path = strings.ReplaceAll(path, "\"", "\"\"")
	command := fmt.Sprintf("ping 127.0.0.1 -n 2 >nul & del /f /q \"%s\"", path)
	cmd := exec.Command("cmd.exe", "/C", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
