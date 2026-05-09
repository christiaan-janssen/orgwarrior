package main

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"time"
)

// Todo represents a single TODO/DONE item parsed from an org file.
type Todo struct {
	Title     string // The headline text after "* TODO" or "* DONE"
	File      string // The org file this task came from
	Line      int    // Line number in the file (1-indexed)
	Level     int    // Number of leading asterisks (depth in the outline)
	Status    string // "TODO" or "DONE"
	Deadline  string // DEADLINE value (e.g. "2026-04-24 Fri")
	Scheduled string // SCHEDULED value (e.g. "2026-05-01 Fri +1w")
	Tags      string // Comma-separated tags parsed from the headline's :tag: group
	Completed string // CLOSED timestamp (e.g. "2026-05-09 Sat 15:04")
}

// Regexes for parsing org-mode headlines and their properties.
var (
	headRe      = regexp.MustCompile(`^(\*+)\s+(.*)`)
	statusRe    = regexp.MustCompile(`\b(TODO|DONE)\b`)
	deadlineRe  = regexp.MustCompile(`DEADLINE:\s*<([^>]+)>`)
	scheduledRe = regexp.MustCompile(`SCHEDULED:\s*<([^>]+)>`)
	closedRe    = regexp.MustCompile(`CLOSED:\s*\[([^\]]+)\]`)
	tagsRe      = regexp.MustCompile(`\s+(:\S+:)\s*$`)
)

// parseTodos reads an org file and returns all TODO and DONE items found.
// DEADLINE, SCHEDULED, and CLOSED are read either inline on the headline or
// from the immediately following non-headline lines.
func parseTodos(path string) ([]Todo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var todos []Todo
	var pending *Todo
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if m := headRe.FindStringSubmatch(line); m != nil {
			if pending != nil {
				todos = append(todos, *pending)
				pending = nil
			}
			if sm := statusRe.FindStringSubmatch(m[2]); sm != nil {
				status := sm[1]
				title := m[2]
				deadline, scheduled, closed := "", "", ""

				if d := deadlineRe.FindStringSubmatch(title); d != nil {
					deadline = d[1]
					title = deadlineRe.ReplaceAllString(title, "")
				}
				if s := scheduledRe.FindStringSubmatch(title); s != nil {
					scheduled = s[1]
					title = scheduledRe.ReplaceAllString(title, "")
				}
				if c := closedRe.FindStringSubmatch(title); c != nil {
					closed = c[1]
					title = closedRe.ReplaceAllString(title, "")
				}

				title = strings.TrimPrefix(title, status+" ")

				tags := ""
				if tg := tagsRe.FindStringSubmatch(title); tg != nil {
					raw := strings.Trim(tg[1], ":")
					tags = strings.ReplaceAll(raw, ":", ", ")
					title = strings.TrimSpace(tagsRe.ReplaceAllString(title, ""))
				}

				pending = &Todo{
					Title:     strings.TrimSpace(title),
					File:      path,
					Line:      lineNum,
					Level:     len(m[1]),
					Status:    status,
					Deadline:  deadline,
					Scheduled: scheduled,
					Tags:      tags,
					Completed: closed,
				}
			}
			continue
		}
		if pending != nil {
			if d := deadlineRe.FindStringSubmatch(line); d != nil {
				pending.Deadline = d[1]
			}
			if s := scheduledRe.FindStringSubmatch(line); s != nil {
				pending.Scheduled = s[1]
			}
			if c := closedRe.FindStringSubmatch(line); c != nil {
				pending.Completed = c[1]
			}
		}
	}
	if pending != nil {
		todos = append(todos, *pending)
	}
	return todos, scanner.Err()
}

// formatOrgDate converts a "YYYY-MM-DD" string into Org-mode's angle-bracket
// format "<YYYY-MM-DD Day>". If parsing fails, it wraps the original string.
func formatOrgDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "<" + dateStr + ">"
	}
	return "<" + t.Format("2006-01-02 Mon") + ">"
}
