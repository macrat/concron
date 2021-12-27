//go:build !windows
// +build !windows

package main

import (
	"os/exec"
	"os/user"
	"strconv"
	"syscall"

	"go.uber.org/zap"
)

var (
	DefaultShell     = "/bin/sh"
	ShellCommandOpts = "-c"
)

// SetUID sets execution user to exec.Cmd.
func SetUID(cmd *exec.Cmd, u *user.User) error {
	if cur, err := user.Current(); err == nil && cur.Uid == u.Uid {
		return nil
	}

	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return err
	}

	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return err
	}

	zap.L().Debug(
		"set uid/gid",
		zap.Uint64("uid", uid),
		zap.Uint64("gid", gid),
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		},
	}

	return nil
}
