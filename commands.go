package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// collectTodos finds, filters, and parses all org files, returning a flat list.
func collectTodos(cfg *Config) ([]Todo, []string) {
	files, err := findOrgFiles(cfg.Paths)
	if err != nil {
		return nil, nil
	}
	var filtered []string
	for _, f := range files {
		if len(cfg.Files) > 0 && !contains(cfg.Files, filepath.Base(f)) {
			continue
		}
		filtered = append(filtered, f)
	}
	var todos []Todo
	for _, f := range filtered {
		t, _ := parseTodos(f)
		todos = append(todos, t...)
	}
	return todos, filtered
}

// handleAdd appends a new TODO task to the default org file.
// It parses due:, sched:, and tags: from the argument list.
func handleAdd(cfg *Config, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: orgwarrior add <title> [due:YYYY-MM-DD] [sched:YYYY-MM-DD] [tags:tag1,tag2]")
		os.Exit(1)
	}

	titleParts := []string{}
	due, sched, tags := "", "", ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "due:") {
			due = strings.TrimPrefix(arg, "due:")
		} else if strings.HasPrefix(arg, "sched:") {
			sched = strings.TrimPrefix(arg, "sched:")
		} else if strings.HasPrefix(arg, "tags:") {
			tags = strings.TrimPrefix(arg, "tags:")
		} else {
			titleParts = append(titleParts, arg)
		}
	}

	title := strings.Join(titleParts, " ")

	orgTags := ""
	if tags != "" {
		parts := strings.Split(tags, ",")
		var cleaned []string
		for _, p := range parts {
			cleaned = append(cleaned, strings.TrimSpace(p))
		}
		orgTags = " :" + strings.Join(cleaned, ":") + ":"
	}

	targetFile := expandPath(cfg.DefaultFile)

	f, err := os.OpenFile(targetFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening %s: %s\n", targetFile, err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Fprintf(f, "\n* TODO %s%s\n", title, orgTags)
	if due != "" || sched != "" {
		line := "  "
		if sched != "" {
			line += "SCHEDULED: " + formatOrgDate(sched) + " "
		}
		if due != "" {
			line += "DEADLINE: " + formatOrgDate(due) + " "
		}
		fmt.Fprintln(f, strings.TrimSpace(line))
	}
	fmt.Printf("Added task to %s\n", targetFile)
}

// handleList prints all TODO items in an aligned table grouped by file.
func handleList(cfg *Config, filterArgs []string) {
	todos, files := collectTodos(cfg)
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no org files found")
		return
	}

	var filtered []Todo
	for _, t := range todos {
		if t.Status == "TODO" {
			filtered = append(filtered, t)
		}
	}
	todos = filtered

	todos = applyFilters(todos, filterArgs)

	if len(todos) == 0 {
		fmt.Fprintln(os.Stderr, "no tasks match")
		return
	}

	idW, titleW, tagsW, schedW, deadW := colWidths(todos)

	pad := "   "
	printHeader := func() {
		fmt.Printf("%-*s%s%-*s%s%-*s%s%-*s%s%s\n", idW, "ID", pad, titleW, "Title", pad, tagsW, "Tags", pad, schedW, "Scheduled", pad, "Deadline")
		fmt.Println(strings.Repeat("-", idW+titleW+tagsW+schedW+deadW+len(pad)*4))
	}

	idx := 1
	headerPrinted := false
	for _, f := range files {
		var ft []Todo
		for _, t := range todos {
			if t.File == f {
				ft = append(ft, t)
			}
		}
		if len(ft) == 0 {
			continue
		}
		if headerPrinted {
			fmt.Println()
		}
		fmt.Printf("%s\n", cyan(filepath.Base(f)))
		if !headerPrinted {
			printHeader()
			headerPrinted = true
		} else {
			printHeader()
		}
		for _, t := range ft {
			sched := dateColor(t.Scheduled)
			dead := dateColor(t.Deadline)
			fmt.Printf("%-*d%s%-*s%s%-*s%s%-*s%s%s\n", idW, idx, pad, titleW, t.Title, pad, tagsW, t.Tags, pad, schedW, sched, pad, dead)
			idx++
		}
	}
}

// handleDone marks a task as complete by index (1-based, from the list output).
func handleDone(cfg *Config, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: orgwarrior done <id>")
		os.Exit(1)
	}

	todos, _ := collectTodos(cfg)
	if len(todos) == 0 {
		fmt.Fprintln(os.Stderr, "no tasks found")
		os.Exit(1)
	}

	id := 0
	for _, arg := range args {
		fmt.Sscanf(arg, "%d", &id)
		break
	}
	if id < 1 || id > len(todos) {
		fmt.Fprintf(os.Stderr, "invalid id %d (range 1-%d)\n", id, len(todos))
		os.Exit(1)
	}

	t := todos[id-1]
	now := time.Now().Format("2006-01-02 Mon 15:04")

	lines, err := readLines(t.File)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %s\n", t.File, err)
		os.Exit(1)
	}

	target := t.Line - 1
	lines[target] = strings.Replace(lines[target], "TODO", "DONE", 1)

	closedLine := fmt.Sprintf("  CLOSED: [%s]", now)
	inserted := false
	for i := target + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "*") {
			lines = append(lines[:i], append([]string{closedLine}, lines[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		lines = append(lines, closedLine)
	}

	if err := writeLines(t.File, lines); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %s\n", t.File, err)
		os.Exit(1)
	}

	fmt.Printf("Done: %s\n", t.Title)
}

// handleModify updates an existing task by ID (1-based, from the list output).
func handleModify(cfg *Config, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: orgwarrior modify <id> [title] [due:YYYY-MM-DD] [sched:YYYY-MM-DD] [tags:tag1,tag2]")
		os.Exit(1)
	}

	id := 0
	fmt.Sscanf(args[0], "%d", &id)
	if id < 1 {
		fmt.Fprintln(os.Stderr, "invalid id")
		os.Exit(1)
	}

	todos, _ := collectTodos(cfg)
	if id > len(todos) {
		fmt.Fprintf(os.Stderr, "invalid id %d (range 1-%d)\n", id, len(todos))
		os.Exit(1)
	}
	t := todos[id-1]

	modArgs := args[1:]
	var titleParts []string
	newDue, newSched, newTags := "", "", ""
	hasDue, hasSched, hasTags := false, false, false
	for _, arg := range modArgs {
		switch {
		case strings.HasPrefix(arg, "due:"):
			newDue = strings.TrimPrefix(arg, "due:")
			hasDue = true
		case strings.HasPrefix(arg, "sched:"):
			newSched = strings.TrimPrefix(arg, "sched:")
			hasSched = true
		case strings.HasPrefix(arg, "tags:"):
			newTags = strings.TrimPrefix(arg, "tags:")
			hasTags = true
		default:
			titleParts = append(titleParts, arg)
		}
	}
	hasTitle := len(titleParts) > 0

	if !hasTitle && !hasDue && !hasSched && !hasTags {
		fmt.Fprintln(os.Stderr, "nothing to modify")
		os.Exit(1)
	}

	lines, err := readLines(t.File)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %s\n", t.File, err)
		os.Exit(1)
	}

	target := t.Line - 1
	hm := headRe.FindStringSubmatch(lines[target])
	if hm == nil {
		fmt.Fprintf(os.Stderr, "could not parse headline\n")
		os.Exit(1)
	}
	stars := hm[1]

	// Current content is "TODO <title> :tags:"
	content := strings.TrimPrefix(hm[2], "TODO ")

	// Extract current tags
	curTags := ""
	if tg := tagsRe.FindStringSubmatch(content); tg != nil {
		curTags = tg[1]
		content = strings.TrimSpace(tagsRe.ReplaceAllString(content, ""))
	}
	curTitle := content

	// Build new headline content
	var newContent string
	if hasTitle {
		newContent = strings.Join(titleParts, " ")
	} else {
		newContent = curTitle
	}

	if hasTags {
		if newTags != "" {
			parts := strings.Split(newTags, ",")
			var cleaned []string
			for _, p := range parts {
				cleaned = append(cleaned, strings.TrimSpace(p))
			}
			newContent += " :" + strings.Join(cleaned, ":") + ":"
		}
	} else if curTags != "" {
		newContent += " " + curTags
	}

	lines[target] = stars + " TODO " + newContent

	// Update DEADLINE/SCHEDULED on following lines
	if hasDue || hasSched {
		foundDue, foundSched := false, false
		propsEnd := target + 1
		for ; propsEnd < len(lines); propsEnd++ {
			trimmed := strings.TrimSpace(lines[propsEnd])
			if trimmed == "" || strings.HasPrefix(trimmed, "*") {
				break
			}
		}

		for i := target + 1; i < propsEnd; i++ {
			if hasDue && deadlineRe.MatchString(lines[i]) {
				if newDue != "" {
					lines[i] = deadlineRe.ReplaceAllString(lines[i], "DEADLINE: "+formatOrgDate(newDue))
				}
				foundDue = true
			}
			if hasSched && scheduledRe.MatchString(lines[i]) {
				if newSched != "" {
					lines[i] = scheduledRe.ReplaceAllString(lines[i], "SCHEDULED: "+formatOrgDate(newSched))
				}
				foundSched = true
			}
		}

		var inserts []string
		if hasSched && !foundSched && newSched != "" {
			inserts = append(inserts, "  SCHEDULED: "+formatOrgDate(newSched))
		}
		if hasDue && !foundDue && newDue != "" {
			inserts = append(inserts, "  DEADLINE: "+formatOrgDate(newDue))
		}
		if len(inserts) > 0 {
			at := target + 1
			lines = append(lines[:at], append(inserts, lines[at:]...)...)
		}
	}

	if err := writeLines(t.File, lines); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %s\n", t.File, err)
		os.Exit(1)
	}

	fmt.Printf("Modified: %s\n", t.Title)
}

// handleDelete removes a task by ID (1-based, from the list output).
func handleDelete(cfg *Config, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: orgwarrior delete <id>")
		os.Exit(1)
	}

	id := 0
	fmt.Sscanf(args[0], "%d", &id)
	if id < 1 {
		fmt.Fprintln(os.Stderr, "invalid id")
		os.Exit(1)
	}

	todos, _ := collectTodos(cfg)
	if id > len(todos) {
		fmt.Fprintf(os.Stderr, "invalid id %d (range 1-%d)\n", id, len(todos))
		os.Exit(1)
	}
	t := todos[id-1]

	lines, err := readLines(t.File)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %s\n", t.File, err)
		os.Exit(1)
	}

	start := t.Line - 1
	end := start + 1
	for ; end < len(lines); end++ {
		trimmed := strings.TrimSpace(lines[end])
		if trimmed == "" || strings.HasPrefix(trimmed, "*") {
			break
		}
	}

	removed := lines[start:end]
	lines = append(lines[:start], lines[end:]...)

	if err := writeLines(t.File, lines); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %s\n", t.File, err)
		os.Exit(1)
	}

	fmt.Printf("Deleted: %s (%d line(s))\n", t.Title, len(removed))
}

// handleCompleted prints recently completed tasks.
func handleCompleted(cfg *Config) {
	todos, files := collectTodos(cfg)
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no org files found")
		return
	}

	lookback := cfg.DoneLookbackDays
	if lookback < 1 {
		lookback = 7
	}
	since := time.Now().AddDate(0, 0, -lookback).Truncate(24 * time.Hour)

	var done []Todo
	for _, t := range todos {
		if t.Status != "DONE" || t.Completed == "" {
			continue
		}
		// Completed format: "2026-05-09 Sat 15:04"
		parts := strings.Fields(t.Completed)
		if len(parts) == 0 {
			continue
		}
		closed, err := time.Parse("2006-01-02", parts[0])
		if err != nil {
			continue
		}
		if closed.Before(since) {
			continue
		}
		done = append(done, t)
	}

	if len(done) == 0 {
		fmt.Println("no recently completed tasks")
		return
	}

	titleW, tagsW, deadW, schedW, compW := 5, 4, 8, 8, 9
	for _, t := range done {
		if len(t.Title) > titleW {
			titleW = len(t.Title)
		}
		if len(t.Tags) > tagsW {
			tagsW = len(t.Tags)
		}
		if len(t.Deadline) > deadW {
			deadW = len(t.Deadline)
		}
		if len(t.Scheduled) > schedW {
			schedW = len(t.Scheduled)
		}
		if len(t.Completed) > compW {
			compW = len(t.Completed)
		}
	}

	pad := "   "
	fmt.Printf("%-*s%s%-*s%s%-*s%s%-*s%s%s\n", titleW, "Title", pad, tagsW, "Tags", pad, schedW, "Scheduled", pad, deadW, "Deadline", pad, "Completed")
	fmt.Println(strings.Repeat("-", titleW+tagsW+schedW+deadW+compW+len(pad)*4))

	idx := 1
	for _, f := range files {
		var fd []Todo
		for _, t := range done {
			if t.File == f {
				fd = append(fd, t)
			}
		}
		if len(fd) == 0 {
			continue
		}
		if idx > 1 {
			fmt.Println()
		}
		fmt.Printf("%s\n", cyan(filepath.Base(f)))
		for _, t := range fd {
			sched := dateColor(t.Scheduled)
			dead := dateColor(t.Deadline)
			fmt.Printf("%-*s%s%-*s%s%-*s%s%-*s%s%s\n", titleW, t.Title, pad, tagsW, t.Tags, pad, schedW, sched, pad, deadW, dead, pad, t.Completed)
			idx++
		}
	}
}

// applyFilters filters a todo slice by tag:, due:before:, due:after:,
// sched:before:, and sched:after: criteria.
func applyFilters(todos []Todo, args []string) []Todo {
	var tagFilter []string
	var dueBefore, dueAfter, schedBefore, schedAfter string

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "tag:"):
			for _, t := range strings.Split(strings.TrimPrefix(arg, "tag:"), ",") {
				tagFilter = append(tagFilter, strings.TrimSpace(t))
			}
		case strings.HasPrefix(arg, "due:before:"):
			dueBefore = strings.TrimPrefix(arg, "due:before:")
		case strings.HasPrefix(arg, "due:after:"):
			dueAfter = strings.TrimPrefix(arg, "due:after:")
		case strings.HasPrefix(arg, "sched:before:"):
			schedBefore = strings.TrimPrefix(arg, "sched:before:")
		case strings.HasPrefix(arg, "sched:after:"):
			schedAfter = strings.TrimPrefix(arg, "sched:after:")
		}
	}

	if len(tagFilter) == 0 && dueBefore == "" && dueAfter == "" && schedBefore == "" && schedAfter == "" {
		return todos
	}

	var result []Todo
	for _, t := range todos {
		if !matchTags(t.Tags, tagFilter) {
			continue
		}
		if !matchDate(t.Deadline, dueBefore, dueAfter) {
			continue
		}
		if !matchDate(t.Scheduled, schedBefore, schedAfter) {
			continue
		}
		result = append(result, t)
	}
	return result
}

// matchTags checks if any of the wanted tags appear in the todo's comma-separated tags.
func matchTags(todoTags string, wanted []string) bool {
	if len(wanted) == 0 {
		return true
	}
	for _, w := range wanted {
		for _, t := range strings.Split(todoTags, ",") {
			if strings.TrimSpace(t) == w {
				return true
			}
		}
	}
	return false
}

// matchDate checks if a date string (e.g. "2026-06-01 Mon") falls within
// the given before/after bounds. Only the date portion (first 10 chars) is compared.
func matchDate(dateStr, before, after string) bool {
	if before == "" && after == "" {
		return true
	}
	if dateStr == "" {
		return false
	}

	dateStr = strings.Fields(dateStr)[0]
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return false
	}

	if before != "" {
		b, err := time.Parse("2006-01-02", before)
		if err != nil || !date.Before(b) {
			return false
		}
	}
	if after != "" {
		a, err := time.Parse("2006-01-02", after)
		if err != nil || !date.After(a) {
			return false
		}
	}
	return true
}

// colWidths computes the max width for each column across all todos.
func colWidths(todos []Todo) (id, title, tags, sched, dead int) {
	id, title, tags, sched, dead = 2, len("Title"), len("Tags"), len("Scheduled"), len("Deadline")
	for i, t := range todos {
		if len(t.Title) > title {
			title = len(t.Title)
		}
		if len(t.Tags) > tags {
			tags = len(t.Tags)
		}
		if len(t.Scheduled) > sched {
			sched = len(t.Scheduled)
		}
		if len(t.Deadline) > dead {
			dead = len(t.Deadline)
		}
		w := len(fmt.Sprintf("%d", i+1))
		if w > id {
			id = w
		}
	}
	return
}

// readLines reads a file into a slice of lines.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines writes a slice of lines back to a file.
func writeLines(path string, lines []string) error {
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// contains reports whether s is in the list.
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
