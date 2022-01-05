//go:build go1.18
// +build go1.18

package main

import (
	"bytes"
	"fmt"
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
		t.Fatalf("failed to parse output of String: %#v: %s", s, err)
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

func FuzzParseCrontab(f *testing.F) {
	for _, bs := range ReadAllTestCrontab(f) {
		for _, parseCommand := range []bool{true, false} {
			for _, enableUserCommand := range []bool{true, false} {
				f.Add(bs, parseCommand, enableUserCommand)
			}
		}
	}

	f.Fuzz(func(t *testing.T, input []byte, parseCommand, enableUserCommand bool) {
		env := Environ{
			fmt.Sprintf("PARSE_COMMAND=%v", parseCommand),
			fmt.Sprintf("ENABLE_USER_COMMAND=%v", enableUserCommand),
		}

		crontab, err := ParseCrontab("/etc/crontab", bytes.NewReader(input), env)
		if err != nil {
			return
		}

		for _, task := range crontab.Tasks {
			CheckTaskSanity(t, task)
		}
	})
}
