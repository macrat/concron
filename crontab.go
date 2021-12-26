package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Crontab is a set of Task.
type Crontab struct {
	Tasks []Task
}

// Has checks the Crontab contains the specified Task.
func (c Crontab) Has(t Task) bool {
	for _, x := range c.Tasks {
		if x.ID == t.ID {
			return true
		}
	}
	return false
}

func (c *Crontab) add(t Task) {
	if !c.Has(t) {
		c.Tasks = append(c.Tasks, t)
	}
}

// ParseCrontab parses crontab file.
func ParseCrontab(path string, r io.Reader, env Environ) (Crontab, error) {
	ct := Crontab{}

	s := bufio.NewScanner(r)
	ln := 0
	for s.Scan() {
		ln++

		line := strings.TrimSpace(s.Text())
		switch DetectLineType(line) {
		case EmptyLine:
			continue
		case TaskLine:
			t, err := ParseTask(path, line, append(Environ{}, env...))
			if err != nil {
				return Crontab{}, fmt.Errorf("%d: %w", ln, err)
			}
			ct.add(t)
		case EnvLine:
			env.Add(line)
		case InvalidLine:
			return Crontab{}, fmt.Errorf("%d: invalid line", ln)
		}
	}
	return ct, s.Err()
}

// LineType is a type of line in crontab file.
type LineType uint8

const (
	InvalidLine LineType = iota
	TaskLine
	EnvLine
	EmptyLine
)

// DetectLineType detects what kind of line is it in crontab file.
func DetectLineType(s string) LineType {
	switch {
	case s == "" || s[0] == byte('#'):
		return EmptyLine
	case strings.ContainsRune("@*0123456789", rune(s[0])):
		return TaskLine
	case strings.ContainsRune(s, '='):
		return EnvLine
	default:
		return InvalidLine
	}
}
