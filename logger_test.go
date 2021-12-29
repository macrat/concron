package main

import (
	"testing"

	"go.uber.org/zap"
)

type TestLogStream struct {
	T *testing.T
}

func (s TestLogStream) Write(p []byte) (int, error) {
	l := string(p)
	if l[len(l)-1] == '\n' {
		l = l[:len(l)-1]
	}
	s.T.Log(l)
	return len(p), nil
}

func (s TestLogStream) Sync() error {
	return nil
}

func NewTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return NewLogger(TestLogStream{t}, zap.DebugLevel)
}
