package main

import (
	"testing"
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
