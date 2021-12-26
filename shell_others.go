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
	ShellCommandFlag = "-c"
)

func SetUID(cmd *exec.Cmd, username string) error {
	u, err := user.Lookup(username)
	if err != nil {
		return err
	}

	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return err
	}

	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return err
	}

	if cur, err := user.Current(); err == nil && cur.Uid == u.Uid {
		return nil
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
