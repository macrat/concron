//go:build !windows
// +build !windows

package main

import (
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

const (
	DefaultShell     = "/bin/sh"
	DefaultShellOpts = "-c"
)

// SetUID sets execution user to exec.Cmd.
func SetUID(_ LoggerHolder, cmd *exec.Cmd, u *user.User) error {
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

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		},
	}

	return nil
}
