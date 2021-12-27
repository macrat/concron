package main

import (
	"bytes"
	"context"
	"io"
	"os/user"
	"runtime"
	"testing"
	"time"
)

func TestParseTask(t *testing.T) {
	tests := []struct {
		Input    string
		Schedule string
		User     string
		Command  string
		Stdin    string
		Bin      string
		Args     string
	}{
		{"@daily  root  echo hello", "@daily", "root", "echo hello", "", "echo", "hello"},
		{"@every 1h  hello  /usr/local/bin/task", "@every 1h", "hello", "/usr/local/bin/task", "", "/usr/local/bin/task", ""},
		{"15 */3 * * *  user  /run.sh", "15 */3 * * *", "user", "/run.sh", "", "/run.sh", ""},
		{" 1   1 * * 1,3\troot\techo  wah", "1 1 * * 1,3", "root", "echo wah", "", "echo", "wah"},
		{"@hourly  root  date +\\%H:\\%M", "@hourly", "root", "date +%H:%M", "", "date", "+\\%H:\\%M"},
		{"@monthly alice cat%hello%world%", "@monthly", "alice", "cat", "hello\nworld\n", "cat", "%hello%world%"},
		{"* 9-18 * * 1-5  bob  cat -n >> out\\%put%\\%hello\\%%world%", "* 9-18 * * 1-5", "bob", "cat -n >> out%put", "%hello%\nworld\n", "cat", "-n >> out\\%put%\\%hello\\%%world%"},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			task, err := ParseTask("/etc/crontab", tt.Input, Environ{})
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
}

func (r *TestTaskReporter) StartTask(t Task) (finish func(int, error), stdout, stderr io.Writer) {
	return func(exitCode int, err error) {
		r.ExitCode = exitCode
		r.Err = err
	}, &r.Output, &r.Output
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

	tests := []RunTest{
		{"*  echo hello world", Environ{}, "hello world\n", 0},
		{"*  exit 1", Environ{}, "", 1},
	}

	if runtime.GOOS == "windows" {
		tests = append(
			tests,
			RunTest{"*  echo hello %someone%", Environ{"someone=world"}, "hello world\r\n", 0},
			RunTest{"*  echo %USER%:%LOGNAME%", Environ{}, u.Username + ":" + u.Username + "\r\n", 0},
			RunTest{"*  pwd", Environ{"HOME=C:\\"}, "C:\\\r\n", 0},
			RunTest{u.Username + "  pwd", Environ{}, u.HomeDir + "\r\n", 0},
			RunTest{"*  echo $env:SHELL", Environ{"SHELL=powershell.exe"}, "powershell.exe\r\n", 0},
		)
	} else {
		tests = append(
			tests,
			RunTest{"*  echo hello $someone", Environ{"someone=world"}, "hello world\n", 0},
			RunTest{"*  echo $USER:$LOGNAME", Environ{}, u.Username + ":" + u.Username + "\n", 0},
			RunTest{"*  pwd", Environ{"HOME=/"}, "/\n", 0},
			RunTest{u.Username + "  pwd", Environ{}, u.HomeDir + "\n", 0},
			RunTest{"*  {printf \"hello \\%s\\n\", $1}%awk%", Environ{"SHELL=awk", "SHELL_OPTS="}, "hello awk\n", 0},
		)
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

			var r TestTaskReporter
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
