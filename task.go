package main

import (
	"context"
	"errors"
	"hash/crc64"
	"io"
	"os/exec"
	"strings"

	"github.com/google/shlex"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var (
	hashTable      = crc64.MakeTable(crc64.ISO)
	ErrInvalidLine = errors.New("invalid line")
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

	var err error
	t.ScheduleSpec, t.User, t.Command, t.Stdin, err = SplitTaskLine(s, env.GetBool("ENABLE_USER_COLUMN"))
	if err != nil {
		return Task{}, err
	}

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
func (t Task) Run(ctx context.Context, sm TaskReporter) {
	finish, stdout, stderr := sm.StartTask(t)

	args := []string{t.Command}
	if t.Env.GetBool("PARSE_COMMAND") {
		if a, err := shlex.Split(t.Command); err == nil {
			args = a
		}
	}

	cmd := exec.CommandContext(
		ctx,
		t.Env.Get("SHELL", DefaultShell),
		append(ShellOpts(t.Env), args...)...,
	)
	cmd.Stdin = strings.NewReader(t.Stdin)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = []string(t.Env)

	if err := SetUserInfo(sm, cmd, t.User); err != nil {
		finish(-1, err)
		return
	}

	err := cmd.Run()
	finish(cmd.ProcessState.ExitCode(), err)
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
		return cmd + " %" + t.EscapedStdin()
	}
}

// String returns string that usable in crontab file.
// In most cases, its output is not enough to re-construct a crontab file, because it is not included the environment variables.
func (t Task) String() string {
	if t.Env.GetBool("ENABLE_USER_COLUMN") {
		return t.ScheduleSpec + "  " + t.User + "  " + t.CommandWithStdin()
	} else {
		return t.ScheduleSpec + "  " + t.CommandWithStdin()
	}
}

// CommandBin returns the first part of the command.
// It is the command name in most cases.
func (t Task) CommandBin() string {
	if t.Command == "" {
		return ""
	}
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
func SplitTaskLine(s string, includeUser bool) (schedule, user, command, stdin string, err error) {
	xs := strings.Fields(s)
	if len(xs) < 2 || (includeUser && len(xs) < 3) {
		return "", "", "", "", ErrInvalidLine
	}

	if s[0] == byte('@') {
		if strings.HasPrefix(s, "@every") {
			if len(xs) < 4 {
				return "", "", "", "", ErrInvalidLine
			}
			schedule = strings.Join(xs[:2], " ")
			xs = xs[2:]
		} else {
			schedule = xs[0]
			xs = xs[1:]
		}
	} else {
		if len(xs) < 6 || (includeUser && len(xs) < 7) {
			return "", "", "", "", ErrInvalidLine
		}
		schedule = strings.Join(xs[:5], " ")
		xs = xs[5:]
	}

	if includeUser {
		user = xs[0]
		command = strings.Join(xs[1:], " ")
	} else {
		user = "*"
		command = strings.Join(xs, " ")
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

	command = strings.TrimSpace(strings.ReplaceAll(command, "\\%", "%"))
	stdin = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(stdin, "\\%", "\r"), "%", "\n"), "\r", "%")
	return
}

// TaskReporter is a interface to StatusMonitor.
type TaskReporter interface {
	StartTask(t Task) (finish func(exitCode int, err error), stdout, stderr io.Writer)
	L() *zap.Logger
}
