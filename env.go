package main

import (
	"os"
	"strconv"
	"strings"
	"unicode"
)

// Environ is the environment variable manager.
type Environ []string

// GetEnviron returns current environment variables.
func GetEnviron() Environ {
	return Environ(os.Environ())
}

// ParseEnv splits an environment variable to key and value.
// If the input string is invalid as an environment variable, it returns empty strings.
func ParseEnv(s string) (key, value string) {
	xs := strings.SplitN(s, "=", 2)
	if len(xs) != 2 {
		return "", ""
	}

	key = strings.TrimSpace(xs[0])
	if !IsValidKey(key) {
		return "", ""
	}

	value = strings.TrimSpace(xs[1])

	if u, err := strconv.Unquote(value); err == nil {
		value = u
	}

	return
}

// IsValidEnv checks if the input string is a valid environment variable or not.
func IsValidEnv(s string) bool {
	key, _ := ParseEnv(s)
	return key != ""
}

// IsValidKey checks if the input string is a valid key for an environment variable.
func IsValidKey(s string) bool {
	if s == "" {
		return false
	}

	for _, x := range s {
		if !unicode.IsGraphic(rune(x)) || unicode.IsSpace(rune(x)) {
			return false
		}
	}

	return true
}

// Add adds new variable to Environ.
// If the Environ has the same key, it will be overridden.
func (e *Environ) Add(s string) {
	k, v := ParseEnv(s)
	if k == "" {
		return
	}

	prefix := k + "="
	for i := range *e {
		if strings.HasPrefix((*e)[i], prefix) {
			if v == "" {
				*e = append((*e)[:i], (*e)[i+1:]...)
			} else {
				(*e)[i] = prefix + v
			}
			return
		}
	}
	*e = append(*e, prefix+v)
}

// Get returns the value for the specified key from this Environ.
// If the Environ has no value for the key, it returns the defaultValue.
func (e Environ) Get(key, defaultValue string) string {
	key = key + "="
	for _, x := range e {
		if strings.HasPrefix(x, key) {
			xs := strings.SplitN(x, "=", 2)
			if xs[1] == "" {
				return defaultValue
			} else {
				return xs[1]
			}
		}
	}
	return defaultValue
}
