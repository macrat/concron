package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version = "HEAD"
	commit  = "unknown"

	DefaultListen = ":8000"
)

func startServer(ctx context.Context, env Environ) {
	var logLevel zapcore.Level
	if err := logLevel.Set(env.Get("CONCRON_LOGLEVEL", "info")); err != nil {
		undo := PrepareLogger(zap.InfoLevel)
		zap.L().Fatal("unknown log level", zap.Error(err))
		undo()
	}

	undo := PrepareLogger(logLevel)
	defer undo()

	address := env.Get("CONCRON_LISTEN", DefaultListen)
	pathes := filepath.SplitList(env.Get("CONCRON_PATH", "/etc/crontab:/etc/cron.d"))

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

func init() {
	flag.Usage = func() {
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  concron [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Environment Variables:")
		fmt.Println("  CONCRON_PATH        List of path to crontab files. (default: " + DefaultPath + ")")
		fmt.Println("  CONCRON_LISTEN      Listen address of dashboard and metrics. (default: " + DefaultListen + ")")
		fmt.Println("  CONCRON_LOGLEVEL    Log level. debug, info, warn, error, or fatal. (default: info)")
		fmt.Println("  TZ                  Timezone for scheduling.")
		fmt.Println("  SHELL               Path to shell to execute command. (default: " + DefaultShell + ")")
		fmt.Println("  SHELL_OPTS          Path to shell to execute command. (default: " + DefaultShellOpts + ")")
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	showHelp := flag.Bool("h", false, "Show help and exit.")
	showVersion := flag.Bool("v", false, "Show version and exit.")
	flag.Parse()

	if *showHelp {
		fmt.Println("Concron - Cron for Container")
		flag.Usage()
		return
	}
	if *showVersion {
		fmt.Printf("Concron %s (%s)\n", version, commit)
		return
	}

	startServer(ctx, GetEnviron())
}
