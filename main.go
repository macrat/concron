package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version = "HEAD"
	commit  = "unknown"

	DefaultListen = ":8000"
)

func startServer(ctx context.Context, logStream zapcore.WriteSyncer, env Environ) {
	var logLevel zapcore.Level
	if err := logLevel.Set(env.Get("CONCRON_LOGLEVEL", "info")); err != nil {
		NewLogger(os.Stdout, zap.InfoLevel).Fatal("unknown log level", zap.Error(err))
	}

	logger := NewLogger(logStream, logLevel)

	address := env.Get("CONCRON_LISTEN", DefaultListen)
	pathes := filepath.SplitList(env.Get("CONCRON_PATH", "/etc/crontab:/etc/cron.d"))

	logger.Info(
		"start concron",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("address", address),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := cron.New(cron.WithLogger((*CronLogger)(logger)))
	sm := NewStatusMonitor(logger)
	cc := NewCrontabCollector(ctx, c, sm, pathes)

	cc.Register(ctx)

	server := &http.Server{}
	defer server.Close()
	go func() {
		err := http.ListenAndServe(address, sm)
		if err != nil {
			logger.Fatal("http server", zap.String("address", address), zap.Error(err))
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
		fmt.Println("  PARSE_COMMAND       Parse command before pass to shell. \"yes\", \"true\", \"1\", \"enable\" to enable. (default: disable)")
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	showHelp := flag.Bool("h", false, "Show help and exit.")
	showVersion := flag.Bool("v", false, "Show version and exit.")
	flag.CommandLine.SetOutput(os.Stdout)
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

	startServer(ctx, os.Stdout, GetEnviron())
}
