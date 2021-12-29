package main

import (
	"testing"
	"time"
)

func TestTaskWithStatus_DurationStr(t *testing.T) {
	tests := []struct {
		Dur time.Duration
		Str string
	}{
		{1234 * time.Nanosecond, "1.23µs"},
		{123 * time.Microsecond, "123µs"},
		{1234 * time.Microsecond, "1.23ms"},
		{123456 * time.Microsecond, "123.46ms"},
		{1234567 * time.Microsecond, "1.23s"},
		{12345678 * time.Microsecond, "12.35s"},
		{125 * 6 * time.Second, "12m30s"},
	}

	for _, tt := range tests {
		t.Run(tt.Dur.String(), func(t *testing.T) {
			ts := TaskWithStatus{TaskStatus: TaskStatus{Duration: tt.Dur}}
			if s := ts.DurationStr(); s != tt.Str {
				t.Errorf("expected %q but got %q", tt.Str, s)
			}
		})
	}
}

func TestStatusMonitor_Status(t *testing.T) {
	sm := NewStatusMonitor(NewTestLogger(t))

	// ---------- load ----------

	source := "/path/to/crontab"
	sm.StartLoad(source)(Crontab{
		Tasks: []Task{
			{ID: 42, Source: source},
		},
	}, nil)

	if status := sm.Status(); len(status) != 1 {
		t.Errorf("unexpected number of status: %v", status)
	} else if status[0].Path != source {
		t.Errorf("unexpected crontab found: %v", status[0])
	} else if len(status[0].Tasks) != 1 {
		t.Errorf("unexpected number of tasks found: %v", status[0].Tasks)
	} else if status[0].Tasks[0].ID != 42 {
		t.Errorf("unexpected ID task found: %v", status[0].Tasks[0].ID)
	} else if status[0].Tasks[0].ExitCodeStr() != "?" {
		t.Errorf("unexpected exit code found: %s", status[0].Tasks[0].ExitCodeStr())
	}

	// ---------- run ----------

	finish, _, _ := sm.StartTask(Task{ID: 42, Source: source})
	finish(1, nil)

	if status := sm.Status(); len(status) != 1 {
		t.Errorf("unexpected number of status: %v", status)
	} else if status[0].Path != source {
		t.Errorf("unexpected crontab found: %v", status[0])
	} else if len(status[0].Tasks) != 1 {
		t.Errorf("unexpected number of tasks found: %v", status[0].Tasks)
	} else if status[0].Tasks[0].ID != 42 {
		t.Errorf("unexpected ID task found: %v", status[0].Tasks[0].ID)
	} else if status[0].Tasks[0].ExitCodeStr() != "1" {
		t.Errorf("unexpected exit code found: %s", status[0].Tasks[0].ExitCodeStr())
	}

	// ---------- reload ----------

	sm.StartLoad(source)(Crontab{
		Tasks: []Task{
			{ID: 123, Source: source},
		},
	}, nil)

	if status := sm.Status(); len(status) != 1 {
		t.Errorf("unexpected number of status: %v", status)
	} else if status[0].Path != source {
		t.Errorf("unexpected crontab found: %v", status[0])
	} else if len(status[0].Tasks) != 1 {
		t.Errorf("unexpected number of tasks found: %v", status[0].Tasks)
	} else if status[0].Tasks[0].ID != 123 {
		t.Errorf("unexpected ID task found: %v", status[0].Tasks[0].ID)
	} else if status[0].Tasks[0].ExitCodeStr() != "?" {
		t.Errorf("unexpected exit code found: %s", status[0].Tasks[0].ExitCodeStr())
	}

	// ---------- unload ----------

	sm.Unloaded(source)

	if status := sm.Status(); len(status) != 0 {
		t.Errorf("unexpected number of status: %v", status)
	}
}
