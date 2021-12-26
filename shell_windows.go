//go:build windows
// +build windows

package main

import (
	"os/exec"
	"os/user"

	"go.uber.org/zap"
)

var (
	DefaultShell     = "C:\\Windows\\powershell.exe"
	ShellCommandFlag = "/c"
)

func SetUID(cmd *exec.Cmd, username string) error {
	u, err := user.Current()
	if err == nil && u.Username != username {
		zap.L().Warn(
			"change the execution user is not supported in Windows",
			zap.String("username", username),
		)
	}

	return nil
}
