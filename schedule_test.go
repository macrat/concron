package main

import (
	"strconv"
	"testing"
	"time"
)

func TestReloadSchedule(t *testing.T) {
	tests := []struct {
		Input  time.Time
		Output time.Time
	}{
		{time.Date(2021, 1, 2, 15, 4, 0, 0, time.UTC), time.Date(2021, 1, 2, 15, 5, 0, 0, time.UTC)},
		{time.Date(2021, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2021, 1, 2, 15, 5, 0, 0, time.UTC)},
		{time.Date(2021, 1, 2, 15, 4, 29, 100, time.UTC), time.Date(2021, 1, 2, 15, 5, 0, 0, time.UTC)},
		{time.Date(2021, 1, 2, 15, 4, 30, 42, time.UTC), time.Date(2021, 1, 2, 15, 5, 0, 0, time.UTC)},
		{time.Date(2021, 1, 2, 15, 4, 42, 90, time.UTC), time.Date(2021, 1, 2, 15, 5, 0, 0, time.UTC)},
		{time.Date(2021, 1, 2, 15, 4, 59, 1, time.UTC), time.Date(2021, 1, 2, 15, 5, 0, 0, time.UTC)},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output := ReloadSchedule{}.Next(tt.Input)
			if output != tt.Output {
				t.Errorf("%s\nexpected: %s\n but got: %s", tt.Input, tt.Output, output)
			}
		})
	}
}
