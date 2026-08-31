//go:build !windows

package game

import "os/exec"

func configureCloudflaredProcess(cmd *exec.Cmd) {}
