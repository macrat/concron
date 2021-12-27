package main

import (
	"fmt"
	"testing"
)

func TestParseEnv(t *testing.T) {
	tests := []struct {
		Input string
		Key   string
		Value string
	}{
		{"hello=world", "hello", "world"},
		{"hello\t= world", "hello", "world"},
		{"hello= world = ", "hello", "world ="},
		{"hello =\" world = \"", "hello", " world = "},
		{"hello = \"world\\\"\"", "hello", "world\""},
		{"hello = ", "hello", ""},
		{" = world", "", ""},
		{"invalid key = world", "", ""},
		{"invalid\x01key = world", "", ""},
		{"invalid string", "", ""},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.Input), func(t *testing.T) {
			key, value := ParseEnv(tt.Input)

			if key != tt.Key {
				t.Errorf("expected key %q but got %q", tt.Key, key)
			}

			if value != tt.Value {
				t.Errorf("expected value %q but got %q", tt.Value, value)
			}
		})
	}
}

func TestIsValidEnv(t *testing.T) {
	tests := []struct {
		Input      string
		IsValidEnv bool
	}{
		{"hello=world", true},
		{"he ll o = wo rl d", false},
		{" hello = world ", true},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			if x := IsValidEnv(tt.Input); x != tt.IsValidEnv {
				t.Errorf("IsValidEnv(%q) expected %v but got %v", tt.Input, tt.IsValidEnv, x)
			}
		})
	}
}

func TestEnviron(t *testing.T) {
	env := Environ{}

	noSuchVariable := "--no-such-key--"

	assert := func(key, want string) {
		t.Helper()

		if actual := env.Get(key, noSuchVariable); actual != want {
			t.Errorf("Get(%q) expected %q but got %q", key, want, actual)
		}
	}

	assertEmpty := func(key, want string) {
		t.Helper()

		if actual := env.GetAllowEmpty(key, noSuchVariable); actual != want {
			t.Errorf("Get(%q) expected %q but got %q", key, want, actual)
		}
	}

	assert("hello", noSuchVariable)
	assert("foo", noSuchVariable)

	env.Set("hello=world")
	assert("hello", "world")
	assertEmpty("hello", "world")

	env.Set("foo = bar ")
	assert("hello", "world")
	assert("foo", "bar")

	env.Set("foo = \" bar \" ")
	assert("hello", "world")
	assert("foo", " bar ")

	env.Set("hello=\"cron\\nworld\"")
	assert("hello", "cron\nworld")
	assert("foo", " bar ")

	env.Set("in valid=hello")
	assert("in valid", noSuchVariable)

	env.Set("hello=")
	assert("hello", noSuchVariable)
	assert("foo", " bar ")
	assertEmpty("hello", "")
	assertEmpty("foo", " bar ")

	env.Set("foo = \"\"")
	assert("hello", noSuchVariable)
	assert("foo", noSuchVariable)
	assertEmpty("hello", "")
	assertEmpty("foo", "")
}
