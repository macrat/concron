package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/user"
	"runtime"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestParseTask(t *testing.T) {
	tests := []struct {
		Input       string
		IncludeUser bool
		Schedule    string
		User        string
		Command     string
		Stdin       string
		Bin         string
		Args        string
	}{
		{"@daily  root  echo hello", true, "@daily", "root", "echo hello", "", "echo", "hello"},
		{"@daily  echo hello", false, "@daily", "*", "echo hello", "", "echo", "hello"},
		{"@every 1h  hello  /usr/local/bin/task", true, "@every 1h", "hello", "/usr/local/bin/task", "", "/usr/local/bin/task", ""},
		{"15 */3 * * *  user  /run.sh", true, "15 */3 * * *", "user", "/run.sh", "", "/run.sh", ""},
		{" 1   1 * * 1,3\troot\techo  wah", true, "1 1 * * 1,3", "root", "echo wah", "", "echo", "wah"},
		{" 1   1 * * 1,3\troot\techo  wah", false, "1 1 * * 1,3", "*", "root echo wah", "", "root", "echo wah"},
		{"@hourly  root  date +\\%H:\\%M", true, "@hourly", "root", "date +%H:%M", "", "date", "+\\%H:\\%M"},
		{"@monthly alice cat%hello%world%", true, "@monthly", "alice", "cat", "hello\nworld\n", "cat", "%hello%world%"},
		{"* 9-18 * * 1-5  bob  cat -n >> out\\%put%\\%hello\\%%world%", true, "* 9-18 * * 1-5", "bob", "cat -n >> out%put", "%hello%\nworld\n", "cat", "-n >> out\\%put %\\%hello\\%%world%"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%v", tt.Input, tt.IncludeUser), func(t *testing.T) {
			task, err := ParseTask("/etc/crontab", tt.Input, Environ{fmt.Sprintf("ENABLE_USER_COLUMN=%v", tt.IncludeUser)})
			if err != nil {
				t.Fatalf("failed to parse: %s", err)
			}

			if task.ScheduleSpec != tt.Schedule {
				t.Errorf("unexpected schedule\nexpected: %q\n but got: %q", tt.Schedule, task.ScheduleSpec)
			}

			if task.User != tt.User {
				t.Errorf("unexpected user\nexpected: %q\n but got: %q", tt.User, task.User)
			}

			if task.Command != tt.Command {
				t.Errorf("unexpected command\nexpected: %q\n but got: %q", tt.Command, task.Command)
			}

			if task.Stdin != tt.Stdin {
				t.Errorf("unexpected stdin\nexpected: %q\n but got: %q", tt.Stdin, task.Stdin)
			}

			if bin := task.CommandBin(); bin != tt.Bin {
				t.Errorf("unexpected bin\nexpected: %q\n but got: %q", tt.Bin, bin)
			}

			if args := task.CommandArgs(); args != tt.Args {
				t.Errorf("unexpected args\nexpected: %q\n but got: %q", tt.Args, args)
			}
		})
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		Input   string
		Command string
		Stdin   string
	}{
		{"echo hello", "echo hello", ""},
		{"date +\\%Y-\\%m-\\%d", "date +%Y-%m-%d", ""},
		{"cat%hello%world", "cat", "hello\nworld"},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			command, stdin := ParseCommand(tt.Input)

			if command != tt.Command {
				t.Errorf("expected command %q but got %q", tt.Command, command)
			}

			if stdin != tt.Stdin {
				t.Errorf("expected stdin %q but got %q", tt.Stdin, stdin)
			}
		})
	}
}

type TestTaskReporter struct {
	Output   bytes.Buffer
	ExitCode int
	Err      error
	Logger   *zap.Logger
}

func (r *TestTaskReporter) StartTask(t Task) (finish func(int, error), stdout, stderr io.Writer) {
	return func(exitCode int, err error) {
		r.ExitCode = exitCode
		r.Err = err
	}, &r.Output, &r.Output
}

func (r *TestTaskReporter) L() *zap.Logger {
	return r.Logger
}

func TestTask_Run(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("failed to get current user info: %s", err)
	}

	type RunTest struct {
		Input    string
		Env      Environ
		Output   string
		ExitCode int
	}

	var tests []RunTest

	if runtime.GOOS == "windows" {
		tests = []RunTest{
			{"@echo hello world", Environ{}, "hello world\r\n", 0},
			{"@exit 1", Environ{}, "", 1},
			{"@echo hello \\%SOMEONE\\%", Environ{"SOMEONE=world"}, "hello world\r\n", 0},
			{"@echo \\%USER\\% \\%LOGNAME\\%", Environ{}, u.Username + " " + u.Username + "\r\n", 0},
			{"*  @echo \\%USER\\% \\%LOGNAME\\%", Environ{"ENABLE_USER_COLUMN=yes"}, u.Username + " " + u.Username + "\r\n", 0},
			{"@cd", Environ{"HOME=C:\\"}, "C:\\\r\n", 0},
			{u.Username + "  @cd", Environ{"ENABLE_USER_COLUMN=enable"}, u.HomeDir + "\r\n", 0},
			{"echo $env:SHELL", Environ{"SHELL=powershell.exe"}, "powershell.exe\r\n", 0},
		}
	} else {
		tests = []RunTest{
			{"echo hello world", Environ{}, "hello world\n", 0},
			{"exit 1", Environ{}, "", 1},
			{"echo hello $someone", Environ{"someone=world"}, "hello world\n", 0},
			{"echo $USER:$LOGNAME", Environ{}, u.Username + ":" + u.Username + "\n", 0},
			{"*  echo $USER:$LOGNAME", Environ{"ENABLE_USER_COLUMN=yes"}, u.Username + ":" + u.Username + "\n", 0},
			{"pwd", Environ{"HOME=/"}, "/\n", 0},
			{u.Username + "  pwd", Environ{"ENABLE_USER_COLUMN=enable"}, u.HomeDir + "\n", 0},
			{"{printf \"hello \\%s\\n\", $1}%awk%", Environ{"SHELL=awk", "SHELL_OPTS="}, "hello awk\n", 0},
			{"10 13", Environ{"SHELL=seq", "SHELL_OPTS=", "PARSE_COMMAND=yes"}, "10\n11\n12\n13\n", 0},
		}
		switch runtime.GOOS {
		case "linux":
			tests = append(
				tests,
				RunTest{"10 14", Environ{"SHELL=seq", "SHELL_OPTS=", "PARSE_COMMAND=no"}, "seq: invalid floating point argument: '10 14'\nTry 'seq --help' for more information.\n", 1},
			)
		case "darwin":
			tests = append(
				tests,
				RunTest{"10 14", Environ{"SHELL=seq", "SHELL_OPTS=", "PARSE_COMMAND=no"}, "seq: invalid floating point argument: 10 14\n", 2},
			)
		}
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Input, func(t *testing.T) {
			t.Parallel()

			task, err := ParseTask("test", "@reboot "+tt.Input, tt.Env)
			if err != nil {
				t.Fatalf("failed to parse task: %s", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			r := TestTaskReporter{Logger: NewTestLogger(t)}
			task.Job(ctx, &r).Run()

			if r.Output.String() != tt.Output {
				t.Errorf("unexpected output\nexpected: %q\n but got: %q", tt.Output, r.Output)
			}

			if r.ExitCode != tt.ExitCode {
				t.Errorf("unexpected exit code: expected %d but got %d", tt.ExitCode, r.ExitCode)
			}
		})
	}
}
