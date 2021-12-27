package main

import (
	"os/exec"
	"os/user"

	"github.com/google/shlex"
)

// ShellOpts gets SHELL_OPTS from Environ and parses it.
func ShellOpts(env Environ) []string {
	raw := env.GetAllowEmpty("SHELL_OPTS", ShellCommandOpts)
	opts, err := shlex.Split(raw)
	if err != nil {
		return []string{raw}
	}
	return opts
}

// SetUserInfo sets execution user settings to exec.Cmd.
// If the username is "*", it sets current user's information.
func SetUserInfo(cmd *exec.Cmd, username string) (err error) {
	var u *user.User
	if username == "*" {
		u, err = user.Current()
		if err != nil {
			return err
		}
	} else {
		u, err = user.Lookup(username)
		if err != nil {
			return err
		}
	}

	e := Environ(cmd.Env)
	e.Set("USER=" + u.Username)
	e.Set("LOGNAME=" + u.Username)
	e.Set("HOME=" + e.Get("HOME", u.HomeDir))
	cmd.Env = []string(e)

	cmd.Dir = e.Get("HOME", cmd.Dir)

	return SetUID(cmd, u)
}
