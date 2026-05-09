package main

import (
	"fmt"
	"os"
	"strings"
)

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

	// Build the Org-mode tag group, e.g. " :work:urgent:".
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

// handleList scans all org files in the configured paths and prints
// TODO items grouped by file in an aligned table.
func handleList(cfg *Config) {
	files, err := findOrgFiles(cfg.Paths)
	if err != nil || len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no org files found")
		return
	}

	// Collect all todos to compute column widths.
	var allTodos []Todo
	for _, f := range files {
		todos, _ := parseTodos(f)
		allTodos = append(allTodos, todos...)
	}

	titleW, tagsW, schedW, deadW := colWidths(allTodos)

	pad := "   "
	printHeader := func() {
		fmt.Printf("%-*s%s%-*s%s%-*s%s%s\n", titleW, "Title", pad, tagsW, "Tags", pad, schedW, "Scheduled", pad, "Deadline")
		fmt.Println(strings.Repeat("-", titleW+tagsW+schedW+deadW+len(pad)*3))
	}

	headerPrinted := false
	for _, f := range files {
		todos, _ := parseTodos(f)
		if len(todos) == 0 {
			continue
		}
		if headerPrinted {
			fmt.Println()
		}
		fmt.Printf("%s\n", f)
		if !headerPrinted {
			printHeader()
			headerPrinted = true
		} else {
			printHeader()
		}
		for _, t := range todos {
			fmt.Printf("%-*s%s%-*s%s%-*s%s%s\n", titleW, t.Title, pad, tagsW, t.Tags, pad, schedW, t.Scheduled, pad, t.Deadline)
		}
	}
}

// colWidths computes the maximum display width for each column across all todos,
// ensuring columns are at least as wide as their headers.
func colWidths(todos []Todo) (title, tags, sched, dead int) {
	title, tags, sched, dead = len("Title"), len("Tags"), len("Scheduled"), len("Deadline")
	for _, t := range todos {
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
	}
	return
}
