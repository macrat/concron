package main

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var (
	version = "HEAD"
	commit  = "unknown"
)

func startServer(ctx context.Context, address string, pathes []string) {
	undo := PrepareLogger(zap.DebugLevel)
	defer undo()

	zap.L().Info(
		"start concron",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("address", address),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := cron.New(cron.WithLogger(CronLogger{}))
	sm := NewStatusManager()
	cc := NewCrontabCollector(ctx, c, sm, pathes)

	cc.Register(ctx)

	http.Handle("/", StatusPage{sm})
	http.Handle("/metrics", promhttp.Handler())
	server := &http.Server{}
	defer server.Close()
	go func() {
		err := http.ListenAndServe(address, nil)
		if err != nil {
			zap.L().Fatal("http server", zap.String("address", address), zap.Error(err))
		}
		cancel()
	}()

	go c.Run()
	<-ctx.Done()
	<-c.Stop().Done()
	server.Shutdown(context.Background())
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := GetEnviron()
	listen := env.Get("CONCRON_LISTEN", ":8000")

	pathes := filepath.SplitList(env.Get("CONCRON_PATH", "/etc/crontab:/etc/cron.d"))

	startServer(ctx, listen, pathes)
}
