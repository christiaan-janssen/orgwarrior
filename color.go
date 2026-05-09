package main

import (
	"os"
	"time"
)

const (
	ansiBold    = "\033[1m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiCyan    = "\033[36m"
	ansiMagenta = "\033[35m"
	ansiReset   = "\033[0m"
)

var useColor = false

func init() {
	info, err := os.Stdout.Stat()
	if err == nil && (info.Mode()&os.ModeCharDevice) != 0 {
		useColor = true
	}
}

func bold(s string) string {
	if !useColor {
		return s
	}
	return ansiBold + s + ansiReset
}

func cyan(s string) string {
	if !useColor {
		return s
	}
	return ansiCyan + s + ansiReset
}

func red(s string) string {
	if !useColor {
		return s
	}
	return ansiRed + s + ansiReset
}

func yellow(s string) string {
	if !useColor {
		return s
	}
	return ansiYellow + s + ansiReset
}

func green(s string) string {
	if !useColor {
		return s
	}
	return ansiGreen + s + ansiReset
}

func magenta(s string) string {
	if !useColor {
		return s
	}
	return ansiMagenta + s + ansiReset
}

// dateColor returns a colorized date string. Green if future/upcoming,
// red if past/overdue.
func dateColor(s string) string {
	if s == "" || !useColor {
		return s
	}
	parts := timeParts(s)
	if parts == "" {
		return s
	}
	t, err := time.Parse("2006-01-02", parts)
	if err != nil {
		return s
	}
	if t.Before(time.Now().Truncate(24 * time.Hour)) {
		return red(s)
	}
	return green(s)
}

// timeParts extracts the first space-separated token from s.
func timeParts(s string) string {
	for i, c := range s {
		if c == ' ' || c == '\t' {
			return s[:i]
		}
	}
	return s
}
