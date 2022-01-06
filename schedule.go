package main

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scheduler is a task scheduler.
// This is a simple wrapper for cron.Cron
type Scheduler struct {
	ctx  context.Context
	cron *cron.Cron
	sm   *StatusMonitor
}

func NewScheduler(ctx context.Context, sm *StatusMonitor) *Scheduler {
	return &Scheduler{
		ctx:  ctx,
		cron: cron.New(cron.WithLogger((*CronLogger)(sm.L()))),
		sm:   sm,
	}
}

// RegisterFunc registers a function to the scheduler.
func (s *Scheduler) RegisterFunc(schedule cron.Schedule, fn func()) cron.EntryID {
	return s.cron.Schedule(schedule, cron.FuncJob(fn))
}

// RegisterTask registers a task to the scheduler.
func (s *Scheduler) RegisterTask(t Task) cron.EntryID {
	return s.RegisterFunc(t.Schedule, func() {
		t.Run(s.ctx, s.sm)
	})
}

// RegisterCrontab registers tasks from a Crontab.
func (s *Scheduler) RegisterCrontab(c Crontab, runRebootTask bool) []cron.EntryID {
	l := s.sm.L().With(zap.String("path", c.Path))
	l.Debug("loading")

	var ids []cron.EntryID

	for _, t := range c.Tasks {
		l.Debug(
			"load",
			zap.String("schedule", t.ScheduleSpec),
			zap.String("user", t.User),
			zap.String("command", t.Command),
			zap.String("stdin", t.Stdin),
		)

		if t.IsReboot {
			if runRebootTask {
				go t.Run(s.ctx, s.sm)
			}
		} else {
			ids = append(ids, s.RegisterTask(t))
		}
	}

	return ids
}

func (s *Scheduler) Unregister(id ...cron.EntryID) {
	for _, x := range id {
		s.cron.Remove(x)
	}
}

// Run runs scheduler.
func (s *Scheduler) Run() {
	s.cron.Run()
}

// Stop stops scheduler.
// The result is a chan to await all tasks closed.
func (s *Scheduler) Stop() <-chan struct{} {
	return s.cron.Stop().Done()
}

// ReloadSchedule is a cron schedule for crontab checking.
// This schedule runs on every minute.
type ReloadSchedule struct{}

// Next implements cron.Schedule.
func (s ReloadSchedule) Next(t time.Time) time.Time {
	return t.Add(time.Duration(60-t.Second())*time.Second - time.Duration(t.Nanosecond())*time.Nanosecond)
}
