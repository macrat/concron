//go:build windows
// +build windows

package main

import (
	"os/user"
	"path/filepath"
	"strings"
)

var DefaultPath = `C:\crontab;C:\cron.d`

func init() {
	u, err := user.Current()
	if err == nil {
		DefaultPath = strings.Join([]string{
			DefaultPath,
			filepath.Join(u.HomeDir, "crontab"),
			filepath.Join(u.HomeDir, "cron.d"),
		}, ";")
	}
}
