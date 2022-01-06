package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version = "HEAD"
	commit  = "unknown"

	DefaultListen = ":8000"
)

func prepareLogger(logStream zapcore.WriteSyncer, env Environ) *zap.Logger {
	var logLevel zapcore.Level

	if err := logLevel.Set(env.Get("CONCRON_LOGLEVEL", "info")); err != nil {
		NewLogger(os.Stdout, zap.InfoLevel).Fatal("unknown log level", zap.Error(err))
	}

	return NewLogger(logStream, logLevel)
}

func startServer(ctx context.Context, logStream zapcore.WriteSyncer, env Environ) (exitCode int) {
	logger := prepareLogger(logStream, env)

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

	sm := NewStatusMonitor(logger)

	server := &http.Server{}
	defer server.Close()
	go func() {
		err := http.ListenAndServe(address, sm)
		if err != nil {
			logger.Fatal("http server", zap.String("address", address), zap.Error(err))
			exitCode = 1
		}
		cancel()
	}()

	s := NewScheduler(ctx, sm)
	NewCrontabCollector(ctx, s, sm, pathes).Register(ctx)

	if envtab := env.Get("CONCRON_CRONTAB", ""); envtab != "" {
		finish := sm.StartLoad("CONCRON_CRONTAB")
		ct, err := ParseCrontab("CONCRON_CRONTAB", strings.NewReader(envtab), env)
		finish(ct, err)
		if err != nil {
			return 2
		}
		s.RegisterCrontab(ct, true)
	}

	go s.Run()
	<-ctx.Done()

	sm.StartTerminating()
	<-s.Stop()

	ctx2, cancel2 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel2()
	server.Shutdown(ctx2)

	return
}

func init() {
	flag.Usage = func() {
		fmt.Println("Concron - Cron for Container")
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
		fmt.Println("  CRON_TZ             Timezone for scheduling.")
		fmt.Println("  SHELL               Path to shell to execute command. (default: " + DefaultShell + ")")
		fmt.Println("  SHELL_OPTS          Path to shell to execute command. (default: " + DefaultShellOpts + ")")
		fmt.Println("  PARSE_COMMAND       Parse command before pass to shell. (default: no)")
		fmt.Println("  ENABLE_USER_COLUMN  Parse and use user column in the crontab file. (default: no)")
	}
}

func runHealthCheck(logStream zapcore.WriteSyncer, env Environ) (exitCode int) {
	listen := env.Get("CONCRON_LISTEN", DefaultListen)
	logger := prepareLogger(logStream, env).With(zap.String("address", listen))

	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		logger.Error("invalid address", zap.Error(err))
		return 1
	}
	if host == "" {
		host = "localhost"
	}

	resp, err := http.Get("http://" + net.JoinHostPort(host, port) + "/livez")
	if err != nil {
		logger.Error("failed to communicate with server", zap.Error(err))
		return 1
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logger.Info(resp.Status, zap.Int("code", resp.StatusCode))
		return 0
	} else {
		logger.Error(resp.Status, zap.Int("code", resp.StatusCode))
		return 1
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	showHelp := flag.Bool("h", false, "Show help and exit.")
	showVersion := flag.Bool("v", false, "Show version and exit.")
	healthCheck := flag.Bool("health-check", false, "Check if Concron running healthy on CONCRON_LISTEN.")
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Parse()

	switch {
	case *showHelp:
		flag.Usage()
	case *showVersion:
		fmt.Printf("Concron %s (%s)\n", version, commit)
	case *healthCheck:
		runHealthCheck(os.Stdout, GetEnviron())
	default:
		os.Exit(startServer(ctx, os.Stdout, GetEnviron()))
	}
}
