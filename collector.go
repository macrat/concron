package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// CrontabCollector is a service to look for crontab files in file storage, and start CrontabWatcher for found ones.
//
// CrontabCollector handles only added files. The changed or removed files handled by CrontabWatcher.
type CrontabCollector struct {
	sync.Mutex

	// Pathes is candidate file names or directory names.
	Pathes []string

	Monitor *StatusMonitor

	cron       *cron.Cron
	lastFounds []string
}

// NewCrontabCollector makes new CrontabCollector.
func NewCrontabCollector(ctx context.Context, c *cron.Cron, sm *StatusMonitor, pathes []string) *CrontabCollector {
	for i := range pathes {
		pathes[i] = filepath.Clean(pathes[i])
	}

	sm.L().Info(
		"search crontab",
		zap.Strings("path", pathes),
	)

	cc := &CrontabCollector{
		Pathes:  pathes,
		Monitor: sm,
		cron:    c,
	}
	cc.searchAndLoad(ctx, true)
	sm.FinishFirstLoad()
	return cc
}

func (c *CrontabCollector) checkFile(ctx context.Context, path string, onReboot bool) error {
	for _, p := range c.lastFounds {
		if p == path {
			return nil
		}
	}

	w, err := NewCrontabWatcher(ctx, c.cron, c.Monitor, path, onReboot)
	if err != nil {
		return err
	}
	w.Register(ctx)

	return nil
}

func (c *CrontabCollector) checkRecursive(ctx context.Context, path string, onReboot bool) (founds []string) {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return
	} else if err != nil {
		return
	}

	if !stat.IsDir() {
		c.checkFile(ctx, path, onReboot)
		return []string{path}
	}

	err = fs.WalkDir(os.DirFS(path), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			p = filepath.Join(path, p)
			err = c.checkFile(ctx, p, onReboot)
			if err == nil {
				founds = append(founds, p)
			}
		}

		return nil
	})
	if err != nil {
		c.Monitor.L().Warn("search crontab", zap.Error(err))
	}

	return
}

func (c *CrontabCollector) searchAndLoad(ctx context.Context, onReboot bool) {
	c.Lock()
	defer c.Unlock()

	founds := []string{}
	for _, p := range c.Pathes {
		files := c.checkRecursive(ctx, p, onReboot)
		founds = append(founds, files...)
	}
	c.lastFounds = founds
}

// Register registers collector task to look for crontab files and start CrontabWatcher if found.
func (c *CrontabCollector) Register(ctx context.Context) {
	c.cron.Schedule(ReloadSchedule{}, cron.FuncJob(func() {
		c.searchAndLoad(ctx, false)
	}))
}
