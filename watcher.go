package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// CrontabWatcher is a service to watch the crontab file and reload if it changed.
type CrontabWatcher struct {
	sync.Mutex

	Path          string
	StatusManager *StatusManager

	cron        *cron.Cron
	modtime     time.Time
	size        int
	entries     []cron.EntryID
	observeTask cron.EntryID
}

func NewCrontabWatcher(ctx context.Context, c *cron.Cron, sm *StatusManager, path string, onReboot bool) (*CrontabWatcher, error) {
	w := &CrontabWatcher{
		Path:          path,
		StatusManager: sm,
		cron:          c,
	}
	err := w.load(ctx, onReboot)
	return w, err
}

func (w *CrontabWatcher) readCrontab() (Crontab, time.Time, error) {
	f, err := os.Open(w.Path)
	if err != nil {
		return Crontab{}, time.Time{}, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return Crontab{}, time.Time{}, err
	}

	ct, err := ParseCrontab(w.Path, f, GetEnviron())
	return ct, stat.ModTime(), err
}

func (w *CrontabWatcher) load(ctx context.Context, onReboot bool) error {
	w.Lock()
	defer w.Unlock()

	finish := w.StatusManager.StartLoad(w.Path)

	ct, modtime, err := w.readCrontab()
	if err != nil {
		finish(ct, err)
		return err
	}

	for _, e := range w.entries {
		w.cron.Remove(e)
	}

	w.entries = []cron.EntryID{}

	l := zap.L().With(zap.String("path", w.Path))
	l.Debug("loading")
	for _, t := range ct.Tasks {
		l.Debug(
			"load",
			zap.String("schedule", t.ScheduleSpec),
			zap.String("command", t.Command),
			zap.String("stdin", t.Stdin),
		)

		if t.IsReboot {
			if onReboot {
				go t.Run(ctx, w.StatusManager)
			}
		} else {
			w.entries = append(w.entries, w.cron.Schedule(t.Schedule, t.Job(ctx, w.StatusManager)))
		}
	}

	finish(ct, nil)

	w.modtime = modtime
	return nil
}

// Register registers watcher task to observe crontab file's changes.
// If the crontab file removed, the watcher automatically unregister itself.
func (w *CrontabWatcher) Register(ctx context.Context) {
	w.observeTask = w.cron.Schedule(ReloadSchedule{}, cron.FuncJob(func() {
		stat, err := os.Stat(w.Path)
		if os.IsNotExist(err) {
			w.Close()
			return
		} else if err != nil {
			zap.L().Error("failed to check crontab", zap.Error(err))
			return
		}

		if stat.ModTime().After(w.modtime) {
			w.load(ctx, false)
		}
	}))
}

// Close unloads all task that registered via this watcher, and wait for to finish all tasks.
func (w *CrontabWatcher) Close() error {
	w.Lock()
	defer w.Unlock()

	w.cron.Remove(w.observeTask)
	for _, e := range w.entries {
		w.cron.Remove(e)
	}
	w.observeTask = 0

	w.StatusManager.Unloaded(w.Path)

	w.entries = []cron.EntryID{}

	return nil
}

// IsActive checks if this watcher is active or not.
func (w *CrontabWatcher) IsActive() bool {
	w.Lock()
	defer w.Unlock()

	return w.observeTask > 0
}
