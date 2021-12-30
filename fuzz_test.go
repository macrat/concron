//go:build go1.18
// +build go1.18

package main

import (
	"bufio"
	"bytes"
	"os"
	"reflect"
	"testing"
)

// CheckTaskSanity tests if a Task seems correct.
func CheckTaskSanity(t *testing.T, task Task) {
	if task.ScheduleSpec == "" {
		t.Errorf("ScheduleSpec is empty")
	}
	if !task.IsReboot && task.Schedule == nil {
		t.Errorf("IsReboot is false but Schedule is also nil")
	}
	if task.IsReboot && task.Schedule != nil {
		t.Errorf("IsReboot is true but Schedule is also not nil")
	}
	if task.User == "" {
		t.Errorf("User is empty")
	}
	if task.Command == "" && task.Stdin == "" {
		t.Errorf("both of Command and Stdin is empty")
	}

	s := task.String()
	if s == "" {
		t.Fatalf("String is empty")
	}

	task2, err := ParseTask(task.Source, s, task.Env)
	if err != nil {
		t.Fatalf("failed to parse output of String: %s", err)
	}

	if !reflect.DeepEqual(task, task2) {
		t.Fatalf("1st parse and 2nd parse is different\n1st: %#v\n2nd: %#v", task, task2)
	}

	if task.CommandBin() != task2.CommandBin() {
		t.Errorf("CommandBin is different\n1st: %q\n2nd: %q", task.CommandBin(), task2.CommandBin())
	}
	if task.CommandArgs() != task2.CommandArgs() {
		t.Errorf("CommandArgs is different\n1st: %q\n2nd: %q", task.CommandArgs(), task2.CommandArgs())
	}
}

func FuzzParseTask(f *testing.F) {
	file, err := os.Open("./testdata/cron.d/fuzz")
	if err != nil {
		f.Fatalf("failed to read initial data: %s", err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		f.Add(scanner.Text())
	}

	f.Fuzz(func(t *testing.T, spec string) {
		task, err := ParseTask("/etc/crontab", spec, Environ{})
		if err != nil {
			return
		}

		CheckTaskSanity(t, task)
	})
}

func FuzzParseCrontab(f *testing.F) {
	for _, bs := range ReadAllTestCrontab(f) {
		f.Add(bs)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		crontab, err := ParseCrontab("/etc/crontab", bytes.NewReader(input), Environ{})
		if err != nil {
			return
		}

		for _, task := range crontab.Tasks {
			CheckTaskSanity(t, task)
		}
	})
}
