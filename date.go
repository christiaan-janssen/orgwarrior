package main

import (
	"strings"
	"time"
)

func goLayout(format string) string {
	switch format {
	case "DD-MM-YYYY":
		return "02-01-2006"
	case "MM-DD-YYYY":
		return "01-02-2006"
	default:
		return "2006-01-02"
	}
}

func displayLayout(format string) string {
	return goLayout(format) + " Mon"
}

func parseDate(s, format string) (time.Time, error) {
	return time.Parse(goLayout(format), s)
}

func formatDateDisplay(s, format string) string {
	if s == "" {
		return ""
	}
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return s
	}
	t, err := time.Parse("2006-01-02", parts[0])
	if err != nil {
		return s
	}
	if len(parts) > 2 {
		return t.Format(goLayout(format)) + " " + parts[2]
	}
	return t.Format(displayLayout(format))
}

func colorDate(rawDate, display string) string {
	if rawDate == "" || !useColor {
		return display
	}
	parts := timeParts(rawDate)
	if parts == "" {
		return display
	}
	t, err := time.Parse("2006-01-02", parts)
	if err != nil {
		return display
	}
	if t.Before(time.Now().Truncate(24 * time.Hour)) {
		return red(display)
	}
	return green(display)
}

func timeParts(s string) string {
	for i, c := range s {
		if c == ' ' || c == '\t' {
			return s[:i]
		}
	}
	return s
}
