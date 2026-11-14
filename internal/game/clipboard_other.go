//go:build !windows

package game

import (
	"bytes"
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// Non-Windows builds shell out to the usual clipboard tools so the game stays
// cgo-free. Missing tools only disable copy/paste; typing still works.

func clipboardCommands(write bool) [][]string {
	if runtime.GOOS == "darwin" {
		if write {
			return [][]string{{"pbcopy"}}
		}
		return [][]string{{"pbpaste"}}
	}
	if write {
		return [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}}
	}
	return [][]string{{"wl-paste", "--no-newline"}, {"xclip", "-selection", "clipboard", "-o"}, {"xsel", "--clipboard", "--output"}}
}

func readClipboardText() (string, error) {
	for _, args := range clipboardCommands(false) {
		if _, err := exec.LookPath(args[0]); err != nil {
			continue
		}
		out, err := exec.Command(args[0], args[1:]...).Output()
		if err == nil {
			return string(out), nil
		}
	}
	return "", errors.New("no clipboard tool available")
}

func writeClipboardText(text string) error {
	for _, args := range clipboardCommands(true) {
		if _, err := exec.LookPath(args[0]); err != nil {
			continue
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = bytes.NewReader([]byte(text))
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return errors.New("no clipboard tool available: install " + strings.Join([]string{"wl-copy", "xclip", "xsel"}, " / "))
}
