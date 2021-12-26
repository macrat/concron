package main

import (
	"time"
)

// ReloadSchedule is a cron schedule for crontab checking.
// This schedule runs on every minute but shifted half a minute.
type ReloadSchedule struct{}

// Next implements cron.Schedule.
func (s ReloadSchedule) Next(t time.Time) time.Time {
	return t.Add(time.Duration(60-t.Second())*time.Second - time.Duration(t.Nanosecond())*time.Nanosecond)
}
