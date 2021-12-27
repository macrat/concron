//go:build windows
// +build windows

package main

import (
	"os/exec"
	"os/user"

	"go.uber.org/zap"
)

var DefaultShell = GetEnviron().Get("COMSPEC", `C:\WINDOWS\System32\cmd.exe`)

const DefaultShellOpts = "/C"

// SetUID sets execution user to exec.Cmd.
// In Windows, this function doesn't set user.
func SetUID(cmd *exec.Cmd, u *user.User) error {
	cur, err := user.Current()
	if err == nil && cur.Uid != u.Uid {
		zap.L().Warn(
			"change the execution user is not supported in Windows",
			zap.String("username", u.Username),
		)
	}

	return nil
}
