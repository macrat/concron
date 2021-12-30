package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectLineType(t *testing.T) {
	tests := []struct {
		Input string
		Type  LineType
	}{
		{"", EmptyLine},
		{"# this is comment", EmptyLine},
		{"* * * * *\troot\techo hello world", TaskLine},
		{"15 */2 * * *\tuser\techo hello world", TaskLine},
		{"@hourly me echo wah", TaskLine},
		{"MAILTO=\"\"", EnvLine},
		{"SHELL = /bin/sh", EnvLine},
		{"INVAlID LINE", InvalidLine},
		{"this is an invalid line", InvalidLine},
	}

	for _, tt := range tests {
		got := DetectLineType(tt.Input)
		if got != tt.Type {
			t.Errorf("%q is %v but got %v", tt.Input, tt.Type, got)
		}
	}
}

func ReadTestCrontab(t testing.TB, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("./testdata/cron.d", name))
	if err != nil {
		t.Fatalf("failed to read crontab: %s", err)
	}

	return raw
}

func ReadAllTestCrontab(t testing.TB) [][]byte {
	t.Helper()

	xs, err := os.ReadDir("./testdata/cron.d")
	if err != nil {
		t.Fatalf("failed to read crontab directory: %s", err)
	}

	var r [][]byte
	for _, x := range xs {
		r = append(r, ReadTestCrontab(t, x.Name()))
	}
	return r
}

func TestParseCrontab(t *testing.T) {
	type TaskTest struct {
		Spec string
		Env  Environ
	}

	tests := []struct {
		Input []byte
		Tasks []TaskTest
	}{
		{
			ReadTestCrontab(t, "valid"),
			[]TaskTest{
				{"@daily  root  echo hello", Environ{"SHELL=sh", "TZ=Asia/Tokyo"}},
				{"0 0 * * *  ec2-user  echo world", Environ{"SHELL=sh", "TZ=UTC"}},
				{"@reboot  admin  cat %concron%initialized!", Environ{"SHELL=sh", "TZ=w h e r e ?"}},
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			ct, err := ParseCrontab("/path/to/crontab", bytes.NewReader(tt.Input), Environ{})
			if err != nil {
				t.Fatalf("failed to parse: %s", err)
			}

			for j, actual := range ct.Tasks {
				wantStr := tt.Tasks[j].Spec
				if wantStr != actual.String() {
					t.Errorf("%d: expected %q but got %q", j, wantStr, actual.String())
				}

				wantEnv := tt.Tasks[j].Env
				if !reflect.DeepEqual(wantEnv, actual.Env) {
					t.Errorf("%d: expected %#v but got %#v", j, wantEnv, actual.Env)
				}
			}
		})
	}
}
