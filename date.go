package main

import (
	"fmt"
	"strconv"
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
	t, err := time.Parse(goLayout(format), s)
	if err == nil {
		return t, nil
	}
	return resolveRelativeDate(s)
}

var weekdayNames = map[string]time.Weekday{
	"sun": time.Sunday, "mon": time.Monday, "tue": time.Tuesday,
	"wed": time.Wednesday, "thu": time.Thursday, "fri": time.Friday,
	"sat": time.Saturday,
}

func resolveRelativeDate(s string) (time.Time, error) {
	now := time.Now().Truncate(24 * time.Hour)
	lower := strings.ToLower(s)

	switch lower {
	case "today":
		return now, nil
	case "tomorrow":
		return now.AddDate(0, 0, 1), nil
	case "yesterday":
		return now.AddDate(0, 0, -1), nil
	}

	if day, ok := weekdayNames[lower]; ok {
		diff := int(day - now.Weekday())
		if diff <= 0 {
			diff += 7
		}
		return now.AddDate(0, 0, diff), nil
	}

	if len(s) >= 3 && (s[0] == '+' || s[0] == '-') {
		num, err := strconv.Atoi(s[1 : len(s)-1])
		if err == nil {
			switch s[len(s)-1] {
			case 'd':
				return now.AddDate(0, 0, num), nil
			case 'w':
				return now.AddDate(0, 0, num*7), nil
			case 'm':
				return now.AddDate(0, num, 0), nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date %q", s)
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
