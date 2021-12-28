package main

import (
	_ "embed"
	"html/template"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	namespace = "concron"
	infoGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "info",
			Help:      "Information about the Concron.",
			ConstLabels: prometheus.Labels{
				"version": version,
				"commit":  commit,
			},
		},
	)
	loadedTaskGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "loaded_tasks_total",
			Help:      "Number of loaded tasks.",
		},
		[]string{"source", "user"},
	)
	runningTaskGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "running_tasks_total",
			Help:      "Number of current running tasks.",
		},
		[]string{"source", "user"},
	)
	startedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "task_started_total",
			Help:      "How many tasks started.",
		},
		[]string{"source", "schedule", "user", "command", "stdin"},
	)
	finishedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "task_finished_total",
			Help:      "How many tasks finished.",
		},
		[]string{"source", "schedule", "user", "command", "stdin", "exit_code"},
	)
	durationSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  namespace,
			Name:       "task_duration_seconds",
			Help:       "A summary of the duration to execute task.",
			MaxAge:     24 * time.Hour,
			Objectives: map[float64]float64{0: 0, 0.25: 0, 0.5: 0, 0.75: 0, 1: 0},
		},
		[]string{"source", "schedule", "user", "command", "stdin", "exit_code"},
	)
	exitCodeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "task_last_exit_code",
			Help:      "The latest exit code of the task.",
		},
		[]string{"source", "schedule", "user", "command", "stdin"},
	)
	loadCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "crontab_load_total",
			Help:      "How many times loaded crontab.",
		},
		[]string{"path", "status"},
	)
	loadDurationSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: namespace,
			Name:      "crontab_load_duration_seconds",
			Help:      "A summary of the duration to load crontab.",
		},
		[]string{"path", "status"},
	)
)

func init() {
	infoGauge.Set(float64(1))
	prometheus.MustRegister(infoGauge)
	prometheus.MustRegister(loadedTaskGauge)
	prometheus.MustRegister(runningTaskGauge)
	prometheus.MustRegister(startedCounter)
	prometheus.MustRegister(finishedCounter)
	prometheus.MustRegister(durationSummary)
	prometheus.MustRegister(exitCodeGauge)
	prometheus.MustRegister(loadCounter)
	prometheus.MustRegister(loadDurationSummary)
}

// TaskStatus is a status of a task execution.
type TaskStatus struct {
	Timestamp time.Time
	Duration  time.Duration
	ExitCode  int
	Log       string
}

// CrontabStatus is a status of a crontab.
type CrontabStatus struct {
	Running int
	Tasks   []Task
}

// StatusMonitor collects task running status, and serve status/metrics pages.
type StatusMonitor struct {
	sync.Mutex

	crontab map[string]*CrontabStatus
	task    map[uint64]TaskStatus
}

func NewStatusMonitor() *StatusMonitor {
	return &StatusMonitor{
		crontab: make(map[string]*CrontabStatus),
		task:    make(map[uint64]TaskStatus),
	}
}

func (sm *StatusMonitor) setCrontabStatus(path string, cs *CrontabStatus) (deleted *CrontabStatus) {
	sm.Lock()
	defer sm.Unlock()

	if ct, ok := sm.crontab[path]; ok {
		deleted = ct
		for _, t := range ct.Tasks {
			if !t.IsReboot || cs == nil {
				delete(sm.task, t.ID)
			}
		}
	}
	if cs == nil {
		delete(sm.crontab, path)
	} else {
		sm.crontab[path] = cs
	}

	return
}

// StartLoad reports started to loading crontab.
// This function returns a function to report the loading completed.
func (sm *StatusMonitor) StartLoad(path string) func(loaded Crontab, err error) {
	stime := time.Now()

	return func(loaded Crontab, err error) {
		duration := time.Since(stime)

		for _, t := range loaded.Tasks {
			loadedTaskGauge.WithLabelValues(path, t.User).Set(0)
		}
		for _, t := range loaded.Tasks {
			loadedTaskGauge.WithLabelValues(path, t.User).Inc()
		}

		l := zap.L().With(
			zap.String("path", path),
			zap.Duration("duration", duration),
			zap.Int("tasks", len(loaded.Tasks)),
		)

		status := "success"
		if err != nil {
			status = "failure"
			l.Error("failed to load", zap.Error(err))
		} else {
			l.Info("loaded")

			sm.setCrontabStatus(path, &CrontabStatus{
				Tasks: loaded.Tasks,
			})
		}

		loadCounter.WithLabelValues(path, status).Inc()
		loadDurationSummary.WithLabelValues(path, status).Observe(duration.Seconds())
	}
}

// Unloaded reports the crontab unloaded.
func (sm *StatusMonitor) Unloaded(path string) {
	deleted := sm.setCrontabStatus(path, nil)

	zap.L().Info(
		"unloaded",
		zap.String("path", path),
		zap.Int("tasks", len(deleted.Tasks)),
	)
}

// StartTask reports a task has started.
// This function returns a function to report the task has finished, and io.Writer for logging.
func (sm *StatusMonitor) StartTask(t Task) (finish func(exitCode int, err error), stdout, stderr io.Writer) {
	sm.Lock()
	if s, ok := sm.crontab[t.Source]; ok {
		s.Running++
		runningTaskGauge.WithLabelValues(t.Source, t.User).Inc()
	}
	sm.Unlock()

	startedCounter.WithLabelValues(t.Source, t.ScheduleSpec, t.User, t.Command, t.Stdin).Inc()

	l := zap.L().With(
		zap.String("source", t.Source),
		zap.String("schedule", t.ScheduleSpec),
		zap.String("user", t.User),
		zap.String("command", t.Command),
		zap.String("stdin", t.Stdin),
	)
	l.Info("start")

	var logRecord strings.Builder
	stdout = io.MultiWriter(&logRecord, NewStdoutLogger(t))
	stderr = io.MultiWriter(&logRecord, NewStderrLogger(t))

	stime := time.Now()

	finish = func(exitCode int, err error) {
		duration := time.Since(stime)

		finishedCounter.WithLabelValues(t.Source, t.ScheduleSpec, t.User, t.Command, t.Stdin, strconv.Itoa(exitCode)).Inc()
		durationSummary.WithLabelValues(t.Source, t.ScheduleSpec, t.User, t.Command, t.Stdin, strconv.Itoa(exitCode)).Observe(duration.Seconds())
		exitCodeGauge.WithLabelValues(t.Source, t.ScheduleSpec, t.User, t.Command, t.Stdin).Set(float64(exitCode))

		l = l.With(zap.Int("exit_code", exitCode), zap.Duration("duration", duration), zap.Error(err))
		if err == nil {
			l.Info("finish")
		} else {
			l.Error("finish")
		}

		log := logRecord.String()
		if log == "" && err != nil {
			log = err.Error()
		}

		sm.Lock()
		if s, ok := sm.crontab[t.Source]; ok && s.Running > 0 {
			s.Running--
			runningTaskGauge.WithLabelValues(t.Source, t.User).Dec()
		}
		sm.task[t.ID] = TaskStatus{
			Timestamp: stime,
			Duration:  duration,
			ExitCode:  exitCode,
			Log:       log,
		}
		sm.Unlock()
	}

	return
}

type TaskWithStatus struct {
	Task
	TaskStatus
}

// TimestampStr returns timestamp in a human readable string.
// If the task is not executed yet, it returns first execution time.
func (ts TaskWithStatus) TimestampStr() string {
	t := ts.Timestamp
	if t.IsZero() {
		t = ts.Schedule.Next(time.Now())
	}
	return humanize.Time(t)
}

// DurationStr returns duration in a human readable string.
func (ts TaskWithStatus) DurationStr() string {
	d := ts.Duration
	switch {
	case d > time.Second:
		d = d.Round(time.Second / 100)
	case d > time.Millisecond:
		d = d.Round(time.Millisecond / 100)
	case d > time.Microsecond:
		d = d.Round(time.Microsecond / 100)
	}
	return d.String()
}

// ExitCodeStr returns exit code in string.
// If the task is not executed yet, it returns "?" instead of number.
func (ts TaskWithStatus) ExitCodeStr() string {
	if ts.Timestamp.IsZero() {
		return "?"
	} else {
		return strconv.Itoa(ts.ExitCode)
	}
}

type StatusSnapshot struct {
	Path  string
	Tasks []TaskWithStatus
}

// Status reports the current status and logs.
func (sm *StatusMonitor) Status() []StatusSnapshot {
	sm.Lock()
	defer sm.Unlock()

	var r []StatusSnapshot

	for path, ct := range sm.crontab {
		ss := StatusSnapshot{Path: path}
		for _, t := range ct.Tasks {
			s, _ := sm.task[t.ID]
			ss.Tasks = append(ss.Tasks, TaskWithStatus{
				Task:       t,
				TaskStatus: s,
			})
		}
		sort.Slice(ss.Tasks, func(i, j int) bool {
			return ss.Tasks[i].Task.String() < ss.Tasks[j].Task.String()
		})
		r = append(r, ss)
	}

	sort.Slice(r, func(i, j int) bool {
		return r[i].Path < r[j].Path
	})

	return r
}

//go:embed templates/status.html
var statusPageTemplateStr string
var statusPageTemplate = template.Must(template.New("status.html").Parse(statusPageTemplateStr))

//go:embed templates/errors.html
var errorPageTemplateStr string
var errorPageTemplate = template.Must(template.New("errors.html").Parse(errorPageTemplateStr))

//go:embed static/icon.svg
var iconSvg []byte

// ServeHTTP implements http.Handler.
func (sm *StatusMonitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		err = errorPageTemplate.Execute(w, "Method not allowed")
	} else {
		switch r.URL.Path {
		case "/favicon.ico":
			w.Header().Set("Content-Type", "image/svg+xml")
			_, err = w.Write(iconSvg)
		case "/metrics":
			promhttp.Handler().ServeHTTP(w, r)
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			err = statusPageTemplate.Execute(w, map[string]interface{}{
				"Status": sm.Status(),
			})
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			err = errorPageTemplate.Execute(w, "Not found")
		}
	}

	if err != nil {
		zap.L().Error(
			"failed to render page",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
	}
}
