package main

import (
	"context"
	"hash/crc64"
	"os/exec"
	"strings"

	"github.com/robfig/cron/v3"
)

var (
	hashTable = crc64.MakeTable(crc64.ISO)
)

// Task is a single task in the crontab.
// The same task always has the same ID.
type Task struct {
	ID           uint64
	Source       string
	ScheduleSpec string
	Schedule     cron.Schedule
	User         string
	Command      string
	Stdin        string
	Env          Environ
	IsReboot     bool
}

// ParseTask parses one line in the crontab and returns Task.
// This function returns error if the schedule spec is wrong, but it don't returns error even if the command is wrong.
func ParseTask(source string, s string, env Environ) (Task, error) {
	t := Task{Source: source, Env: env}

	t.ScheduleSpec, t.User, t.Command, t.Stdin = SplitTaskLine(s)

	if t.ScheduleSpec == "@reboot" {
		t.IsReboot = true
	} else {
		var err error
		tz := env.Get("CRON_TZ", env.Get("TZ", ""))
		t.Schedule, err = cron.ParseStandard("CRON_TZ=" + tz + " " + t.ScheduleSpec)
		if err != nil {
			return Task{}, err
		}
	}

	id := crc64.New(hashTable)
	id.Write([]byte(strings.Join([]string{
		source,
		t.ScheduleSpec,
		t.User,
		t.Command,
		t.Stdin,
	}, "\n")))
	id.Write([]byte("\n"))
	for _, e := range env {
		id.Write([]byte(e + "\n"))
	}
	t.ID = id.Sum64()

	return t, nil
}

// Run runs the task.
func (t Task) Run(ctx context.Context, sm *StatusManager) {
	finish, stdout, stderr := sm.StartTask(t)

	cmd := exec.CommandContext(ctx, t.Env.Get("SHELL", DefaultShell), t.Env.Get("SHELL_FLAG", ShellCommandFlag), t.Command)
	cmd.Stdin = strings.NewReader(t.Stdin)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = []string(t.Env)

	if t.User != "*" {
		if err := SetUID(cmd, t.User); err != nil {
			finish(-1, err)
			return
		}
	}

	err := cmd.Run()
	finish(cmd.ProcessState.ExitCode(), err)
}

// Job returns cron.Job
func (t Task) Job(ctx context.Context, sm *StatusManager) cron.Job {
	return cron.FuncJob(func() {
		t.Run(ctx, sm)
	})
}

// EscapedStdin is Stdin but escaped % and \n.
func (t Task) EscapedStdin() string {
	return strings.ReplaceAll(strings.ReplaceAll(t.Stdin, "%", "\\%"), "\n", "%")
}

// CommandWithStdin returns the command part in the crontab spec.
func (t Task) CommandWithStdin() string {
	cmd := strings.ReplaceAll(t.Command, "%", "\\%")
	if t.Stdin == "" {
		return cmd
	} else {
		return cmd + "%" + t.EscapedStdin()
	}
}

// String returns string that usable in crontab file.
func (t Task) String() string {
	return t.ScheduleSpec + "  " + t.User + "  " + t.CommandWithStdin()
}

// CommandBin returns the first part of the command.
// It is the command name in most cases.
func (t Task) CommandBin() string {
	return strings.ReplaceAll(strings.Fields(t.Command)[0], "%", "\\%")
}

// CommandArgs returns the command without the first part.
// It is the arguments for command in most cases.
// It includes stdin part.
func (t Task) CommandArgs() string {
	if len(strings.Fields(t.Command)) != 1 {
		return strings.Join(strings.Fields(t.CommandWithStdin())[1:], " ")
	}
	if t.Stdin != "" {
		return "%" + t.EscapedStdin()
	}
	return ""
}

// SplitTaskLine splits a task line in crontab.
func SplitTaskLine(s string) (schedule, user, command, stdin string) {
	xs := strings.Fields(s)

	if s[0] == byte('@') {
		if strings.HasPrefix(s, "@every") {
			schedule = strings.Join(xs[:2], " ")
			user = xs[2]
			command = strings.Join(xs[3:], " ")
		} else {
			schedule = xs[0]
			user = xs[1]
			command = strings.Join(xs[2:], " ")
		}
	} else {
		schedule = strings.Join(xs[:5], " ")
		user = xs[5]
		command = strings.Join(xs[6:], " ")
	}

	command, stdin = ParseCommand(command)
	return
}

// ParseCommand parses a command text in a line of crontab.
func ParseCommand(s string) (command, stdin string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && (i == 0 || s[i-1] != '\\') {
			command = s[:i]
			stdin = s[i+1:]
			break
		}
	}
	if command == "" && stdin == "" {
		command = s
	}

	command = strings.ReplaceAll(command, "\\%", "%")
	stdin = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(stdin, "\\%", "\r"), "%", "\n"), "\r", "%")
	return
}
