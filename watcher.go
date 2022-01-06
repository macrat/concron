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

	Path    string
	Monitor *StatusMonitor

	scheduler   *Scheduler
	modtime     time.Time
	size        int
	entries     []cron.EntryID
	observeTask cron.EntryID
}

func NewCrontabWatcher(ctx context.Context, s *Scheduler, sm *StatusMonitor, path string, onReboot bool) (*CrontabWatcher, error) {
	w := &CrontabWatcher{
		Path:      path,
		Monitor:   sm,
		scheduler: s,
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

	finish := w.Monitor.StartLoad(w.Path)

	ct, modtime, err := w.readCrontab()
	if err != nil {
		finish(ct, err)
		return err
	}

	w.scheduler.Unregister(w.entries...)
	w.entries = w.scheduler.RegisterCrontab(ct, onReboot)

	finish(ct, nil)

	w.modtime = modtime
	return nil
}

// Register registers watcher task to observe crontab file's changes.
// If the crontab file removed, the watcher automatically unregister itself.
func (w *CrontabWatcher) Register(ctx context.Context) {
	w.observeTask = w.scheduler.RegisterFunc(ReloadSchedule{}, func() {
		stat, err := os.Stat(w.Path)
		if os.IsNotExist(err) {
			w.Close()
			return
		} else if err != nil {
			w.Monitor.L().Error("failed to check crontab", zap.Error(err))
			return
		}

		if stat.ModTime().After(w.modtime) {
			w.load(ctx, false)
		}
	})
}

// Close unloads all task that registered via this watcher, and wait for to finish all tasks.
func (w *CrontabWatcher) Close() error {
	w.Lock()
	defer w.Unlock()

	w.scheduler.Unregister(w.observeTask)
	w.scheduler.Unregister(w.entries...)
	w.observeTask = 0

	w.Monitor.Unloaded(w.Path)

	w.entries = []cron.EntryID{}

	return nil
}
