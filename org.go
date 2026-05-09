package main

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"time"
)

// Todo represents a single TODO item parsed from an org file.
type Todo struct {
	Title     string // The headline text after "* TODO"
	File      string // The org file this task came from
	Line      int    // Line number in the file (1-indexed)
	Level     int    // Number of leading asterisks (depth in the outline)
	Deadline  string // DEADLINE value (e.g. "2026-04-24 Fri")
	Scheduled string // SCHEDULED value (e.g. "2026-05-01 Fri +1w")
	Tags      string // Comma-separated tags parsed from the headline's :tag: group
}

// Regexes for parsing org-mode headlines and their properties.
var (
	headRe      = regexp.MustCompile(`^(\*+)\s+(.*)`)
	todoRe      = regexp.MustCompile(`\bTODO\b`)
	deadlineRe  = regexp.MustCompile(`DEADLINE:\s*<([^>]+)>`)
	scheduledRe = regexp.MustCompile(`SCHEDULED:\s*<([^>]+)>`)
	tagsRe      = regexp.MustCompile(`\s+(:\S+:)\s*$`)
)

// parseTodos reads an org file and returns all TODO items found.
// DEADLINE and SCHEDULED are read either inline on the headline or
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
			// On encountering a new headline, save any pending todo.
			if pending != nil {
				todos = append(todos, *pending)
				pending = nil
			}
			if todoRe.MatchString(m[2]) {
				title := m[2]
				deadline, scheduled := "", ""

				// Extract inline DEADLINE and SCHEDULED from the title.
				if d := deadlineRe.FindStringSubmatch(title); d != nil {
					deadline = d[1]
					title = deadlineRe.ReplaceAllString(title, "")
				}
				if s := scheduledRe.FindStringSubmatch(title); s != nil {
					scheduled = s[1]
					title = scheduledRe.ReplaceAllString(title, "")
				}

				title = strings.TrimPrefix(title, "TODO ")

				// Extract trailing Org tags like :work:urgent:.
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
					Deadline:  deadline,
					Scheduled: scheduled,
					Tags:      tags,
				}
			}
			continue
		}
		// If we have a pending todo and this is not a headline, check for
		// DEADLINE or SCHEDULED on the following lines.
		if pending != nil {
			if d := deadlineRe.FindStringSubmatch(line); d != nil {
				pending.Deadline = d[1]
			}
			if s := scheduledRe.FindStringSubmatch(line); s != nil {
				pending.Scheduled = s[1]
			}
		}
	}
	// Flush the last pending todo at EOF.
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
