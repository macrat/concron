package main

import (
	"context"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func Test_reboot(t *testing.T) {
	timeout := 100 * time.Millisecond
	if runtime.GOOS == "windows" {
		timeout = 1 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	u, err := user.Current()
	if err != nil {
		t.Fatalf("failed to get current user information: %s", err)
	}

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "cron.d"), 0755); err != nil {
		t.Fatalf("failed to prepare cron.d: %s", err)
	}

	t.Setenv("CONCRON_TEMP_DIR", dir)

	type TestCrontab struct {
		Path    string
		Content string
	}
	var crontabs []TestCrontab
	if runtime.GOOS == "windows" {
		crontabs = []TestCrontab{
			{"crontab", "@reboot  echo hello world>\\%CONCRON_TEMP_DIR\\%/hello"},
			{filepath.Join("cron.d", "a"), "SHELL=powershell.exe\r\nENABLE_USER_COLUMN=yes\r\n@reboot  " + u.Username + "  Write-Output \"this`r`nis`r`nA\" | Out-File -FilePath $env:CONCRON_TEMP_DIR/a -Encoding ascii -NoNewline"},
		}
	} else {
		crontabs = []TestCrontab{
			{"crontab", "@reboot  echo hello world > $CONCRON_TEMP_DIR/hello"},
			{filepath.Join("cron.d", "a"), "ENABLE_USER_COLUMN=yes\n@reboot  " + u.Username + "  cat > $CONCRON_TEMP_DIR/a%this%is%A"},
		}
	}
	for _, tt := range crontabs {
		if err := os.WriteFile(filepath.Join(dir, tt.Path), []byte(tt.Content), 0644); err != nil {
			t.Fatalf("failed to prepare %s: %s", tt.Path, err)
		}
	}

	startServer(ctx, TestLogStream{t}, Environ{
		"CONCRON_LOGLEVEL=debug",
		"CONCRON_LISTEN=localhost:0",
		"CONCRON_PATH=" + strings.Join([]string{
			filepath.Join(dir, "crontab"),
			filepath.Join(dir, "cron.d"),
		}, string(filepath.ListSeparator)),
	})

	outputs := []struct {
		Path    string
		Content string
	}{
		{"hello", "hello world\n"},
		{"a", "this\nis\nA"},
	}
	for _, tt := range outputs {
		bs, err := os.ReadFile(filepath.Join(dir, tt.Path))
		if err != nil {
			t.Fatalf("failed to read %s: %s", tt.Path, err)
		}
		if runtime.GOOS == "windows" {
			tt.Content = strings.ReplaceAll(tt.Content, "\n", "\r\n")
		}
		if string(bs) != tt.Content {
			t.Errorf("unexpected content of %s\nexpected: %q\n but got: %q", tt.Path, tt.Content, string(bs))
		}
	}
}

func Test_CONCRON_CRONTAB(t *testing.T) {
	timeout := 100 * time.Millisecond
	if runtime.GOOS == "windows" {
		timeout = 1 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dir := t.TempDir()
	env := Environ{
		"CONCRON_LOGLEVEL=debug",
		"CONCRON_LISTEN=localhost:0",
		"CONCRON_PATH=" + strings.Join([]string{
			filepath.Join(dir, "crontab"),
			filepath.Join(dir, "cron.d"),
		}, string(filepath.ListSeparator)),
		"CONCRON_TEMP_DIR=" + dir,
	}
	if runtime.GOOS == "windows" {
		env.Set(`CONCRON_CRONTAB=@reboot echo hello world > \%CONCRON_TEMP_DIR\%\hello`)
	} else {
		env.Set(`CONCRON_CRONTAB=@reboot echo hello world > $CONCRON_TEMP_DIR/hello`)
	}

	startServer(ctx, TestLogStream{t}, env)

	bs, err := os.ReadFile(filepath.Join(dir, "hello"))
	if err != nil {
		t.Fatalf("failed to read output: %s", err)
	}
	expect := "hello world\n"
	if runtime.GOOS == "windows" {
		expect = strings.ReplaceAll(expect, "\n", "\r\n")
	}
	if string(bs) != expect {
		t.Errorf("unexpected content of output\nexpected: %q\n but got: %q", expect, string(bs))
	}
}
