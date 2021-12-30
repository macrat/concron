//go:build fulltest && linux
// +build fulltest,linux

package main

import (
	"context"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"
)

func Test_fulltest(t *testing.T) {
	t.Log("full test started (this test is very slow)")

	u, err := user.Current()
	if err != nil {
		t.Fatalf("failed to get current user information: %s", err)
	}

	tmpls, err := template.ParseGlob("testdata/fulltest/*")
	if err != nil {
		t.Fatalf("failed to read crontab template: %s", err)
	}

	dir := t.TempDir()
	cronPath := filepath.Join(dir, "cron.d")
	outputPath := filepath.Join(dir, "output")

	for _, d := range []string{cronPath, outputPath} {
		if err = os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("failed to make %s directory: %s", d, err)
		}
	}

	for _, tmpl := range tmpls.Templates() {
		f, err := os.Create(filepath.Join(cronPath, tmpl.Name()))
		if err != nil {
			t.Fatalf("failed to create crontab: %s", err)
		}

		err = tmpl.ExecuteTemplate(f, tmpl.Name(), map[string]string{
			"User":       u.Username,
			"CronPath":   cronPath,
			"OutputPath": outputPath,
		})
		if err != nil {
			t.Fatalf("failed to generate crontab: %s", err)
		}
	}

	// wait 20sec if current second is under 10.
	if curSec := time.Now().Second(); curSec < 10 {
		time.Sleep(20 * time.Second)
	}
	// wait 30sec if current second is over 50.
	if curSec := time.Now().Second(); curSec > 50 {
		time.Sleep(30 * time.Second)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute+10*time.Second)
	defer cancel()

	startServer(ctx, TestLogStream{t}, Environ{
		"CONCRON_LOGLEVEL=debug",
		"CONCRON_LISTEN=localhost:8000",
		"CONCRON_PATH=" + cronPath,
	})

	readFile := func(name string) string {
		raw, err := os.ReadFile(filepath.Join(outputPath, name))
		if err != nil {
			t.Errorf("failed to read %s: %s", name, err)
			return ""
		}
		return string(raw)
	}

	readTimingOutput := func(name string) (string, int, int) {
		xs := strings.Split(strings.Trim(readFile(name), "\n"), ":")
		m, err := strconv.Atoi(xs[1])
		if err != nil {
			t.Errorf("failed to parse minute part of %s: %s", name, err)
			return "", 0, 0
		}
		s, err := strconv.Atoi(xs[2])
		if err != nil {
			t.Errorf("failed to parse second part of %s: %s", name, err)
			return "", 0, 0
		}
		return xs[0], m, s
	}

	curMin := time.Now().Minute()

	if key, m, _ := readTimingOutput("reboot"); key != "R" {
		t.Errorf("expected %q but got %q", "", key)
	} else if (m+3)%60 != curMin {
		expect := curMin - 3
		if expect < 0 {
			expect = (expect + 60) % 60
		}
		t.Errorf("unexpected minute: expected %d but got %d", expect, m)
	}

	if key, m, s := readTimingOutput("minutely-a"); key != "A" {
		t.Errorf("expected %q but got %q", "", key)
	} else if (m+1)%60 != curMin {
		expect := curMin - 1
		if expect < 0 {
			expect = 59
		}
		t.Errorf("unexpected minute: expected %d but got %d", expect, m)
	} else if s != 0 {
		t.Errorf("unexpected second: expected %d but got %d", 0, s)
	}

	if key, m, s := readTimingOutput("minutely-b"); key != "B" {
		t.Errorf("expected %q but got %q", "", key)
	} else if m != curMin {
		t.Errorf("unexpected minute: expected %d but got %d", curMin, m)
	} else if s != 0 {
		t.Errorf("unexpected second: expected %d but got %d", 0, s)
	}

	for _, tt := range [][]string{
		{"count3", strings.Repeat("hello world\n", 3)},
		{"count2", strings.Repeat("hello world\n", 2)},
		{"count1", strings.Repeat("hello world\n", 1)},
	} {
		if output := readFile(tt[0]); output != tt[1] {
			t.Errorf("expected %q but got %q", tt[1], output)
		}
	}
}
