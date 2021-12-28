package main

import (
	"io"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	LogStream zapcore.WriteSyncer = os.Stdout
)

// PrepareLogger makes logger and register to zap.L().
// This function returns a function to reset logger to default.
func PrepareLogger(level zapcore.Level) func() {
	conf := zap.NewProductionEncoderConfig()
	conf.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(conf),
		zapcore.Lock(LogStream),
		level,
	))
	undo := zap.ReplaceGlobals(logger)
	return func() {
		undo()
		logger.Sync()
	}
}

// CronLogger is a log wrapper for github.com/robfig/cron/v3.
type CronLogger struct{}

func (_ CronLogger) filterFields(kvs []interface{}) []interface{} {
	rs := []interface{}{}
	for i := 0; i < len(kvs); i += 2 {
		if s, ok := kvs[i].(string); ok && s != "now" {
			rs = append(rs, kvs[i], kvs[i+1])
		}
	}
	return rs
}

// Info implements cron.Logger.
func (l CronLogger) Info(msg string, kvs ...interface{}) {
	zap.L().With(zap.String("task", msg)).Sugar().Debugw("cron", l.filterFields(kvs)...)
}

// Error implements cron.Logger.
func (l CronLogger) Error(err error, msg string, kvs ...interface{}) {
	zap.L().With(zap.String("task", msg), zap.Error(err)).Sugar().Errorw("cron", l.filterFields(kvs)...)
}

// OutputLogger is a logger for hook command output.
// This struct implements io.Writer.
type OutputLogger struct {
	Label  string
	Logger func(string, ...zap.Field)
}

// Write implements io.Writer.
func (l OutputLogger) Write(w []byte) (int, error) {
	s := strings.ReplaceAll(strings.ReplaceAll(string(w), "\r\n", "\n"), "\r", "\n")
	l.Logger("output", zap.String(l.Label, s))

	return len(w), nil
}

// NewStdoutLogger makes a new OutputLogger for stdout.
func NewStdoutLogger(t Task) io.Writer {
	return OutputLogger{
		"stdout",
		zap.L().With(
			zap.String("source", t.Source),
			zap.String("schedule", t.ScheduleSpec),
			zap.String("user", t.User),
			zap.String("command", t.Command),
			zap.String("stdin", t.Stdin),
		).Info,
	}
}

// NewStderrLogger makes a new OutputLogger for stderr.
func NewStderrLogger(t Task) io.Writer {
	return OutputLogger{
		"stderr",
		zap.L().With(
			zap.String("source", t.Source),
			zap.String("schedule", t.ScheduleSpec),
			zap.String("user", t.User),
			zap.String("command", t.Command),
			zap.String("stdin", t.Stdin),
		).Error,
	}
}
