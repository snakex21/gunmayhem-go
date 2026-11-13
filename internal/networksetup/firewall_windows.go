//go:build windows

package networksetup

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

const configureArg = "--gunmayhem-configure-network"

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

func HandleSpecialMode(args []string) (bool, error) {
	if len(args) != 1 || args[0] != configureArg {
		return false, nil
	}
	return true, configureFirewall()
}

func EnsureConfigured() {
	executablePath, err := os.Executable()
	if err == nil && firewallConfigured(executablePath) {
		return
	}
	_ = launchElevatedHelper()
}

func firewallConfigured(executablePath string) bool {
	for _, rule := range firewallRules(executablePath) {
		if !firewallRuleExists(rule) {
			return false
		}
	}
	return true
}

func firewallRuleExists(rule firewallRule) bool {
	cmd := exec.Command("netsh", firewallShowArgs(rule.name)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return firewallOutputHasProgram(out, rule.program)
}

func configureFirewall() error {
	executablePath, err := os.Executable()
	if err != nil {
		return err
	}
	for _, rule := range firewallRules(executablePath) {
		if err := replaceRule(rule); err != nil {
			return err
		}
	}
	return nil
}

func replaceRule(rule firewallRule) error {
	deleteCmd := exec.Command("netsh", firewallDeleteArgs(rule.name)...)
	deleteCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = deleteCmd.Run()

	addCmd := exec.Command("netsh", firewallAddArgs(rule)...)
	addCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("firewall rule %q: %w (%s)", rule.name, err, bytes.TrimSpace(out))
	}
	return nil
}

func launchElevatedHelper() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	params, _ := syscall.UTF16PtrFromString(configureArg)
	dir, _ := syscall.UTF16PtrFromString(filepath.Dir(exe))
	result, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		uintptr(unsafe.Pointer(dir)),
		0,
	)
	if result <= 32 {
		if callErr != syscall.Errno(0) {
			return callErr
		}
		return fmt.Errorf("ShellExecuteW failed: %d", result)
	}
	return nil
}
