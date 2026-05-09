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
func handleList(cfg *Config) {
	todos, files := collectTodos(cfg)
	if len(todos) == 0 || len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no org files found")
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
		ft, _ := parseTodos(f)
		if len(ft) == 0 {
			continue
		}
		if headerPrinted {
			fmt.Println()
		}
		fmt.Printf("%s\n", filepath.Base(f))
		if !headerPrinted {
			printHeader()
			headerPrinted = true
		} else {
			printHeader()
		}
		for _, t := range ft {
			fmt.Printf("%-*d%s%-*s%s%-*s%s%-*s%s%s\n", idW, idx, pad, titleW, t.Title, pad, tagsW, t.Tags, pad, schedW, t.Scheduled, pad, t.Deadline)
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
