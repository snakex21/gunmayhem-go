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

const (
	lanGameplayPort  = 7777
	lanDiscoveryPort = 7778
)

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
	ruleDiscovery     = "Gun Mayhem LAN Discovery"
	ruleGameplay      = "Gun Mayhem LAN Gameplay"
)

func HandleSpecialMode(args []string) (bool, error) {
	if len(args) != 1 || args[0] != configureArg {
		return false, nil
	}
	return true, configureFirewall()
}

func EnsureConfigured() {
	if firewallConfigured() {
		return
	}
	_ = launchElevatedHelper()
}

func firewallConfigured() bool {
	return firewallRuleExists(ruleDiscovery) && firewallRuleExists(ruleGameplay)
}

func firewallRuleExists(name string) bool {
	cmd := exec.Command("netsh", "advfirewall", "firewall", "show", "rule", "name="+name)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return bytes.Contains(bytes.ToLower(out), bytes.ToLower([]byte(name)))
}

func configureFirewall() error {
	if err := replaceRule(ruleDiscovery, "UDP", lanDiscoveryPort); err != nil {
		return err
	}
	if err := replaceRule(ruleGameplay, "TCP", lanGameplayPort); err != nil {
		return err
	}
	return nil
}

func replaceRule(name, protocol string, port int) error {
	deleteCmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name="+name)
	deleteCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = deleteCmd.Run()

	addCmd := exec.Command(
		"netsh", "advfirewall", "firewall", "add", "rule",
		"name="+name,
		"dir=in",
		"action=allow",
		"protocol="+protocol,
		fmt.Sprintf("localport=%d", port),
		"profile=any",
		"remoteip=LocalSubnet",
		"enable=yes",
	)
	addCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("firewall rule %q: %w (%s)", name, err, bytes.TrimSpace(out))
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
