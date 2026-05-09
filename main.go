package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	Paths       []string `json:"paths"`
	DefaultFile string   `json:"default_file"`
}

func defaultConfigPath() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "org-cli", "config.json")
	}
	return ""
}

func loadOrCreateConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		var cfg Config
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}
		if cfg.DefaultFile == "" {
			cfg.DefaultFile = "~/org/inbox.org"
		}
		return &cfg, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	cfg := &Config{
		Paths:       []string{"~/org/"},
		DefaultFile: "~/org/inbox.org",
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	f, err = os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func findOrgFiles(paths []string) ([]string, error) {
	var files []string
	for _, p := range paths {
		p = expandPath(p)
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		filepath.WalkDir(p, func(fpath string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(fpath, ".org") {
				files = append(files, fpath)
			}
			return nil
		})
	}
	return files, nil
}

type Todo struct {
	Title     string
	File      string
	Level     int
	Deadline  string
	Scheduled string
	Tags      string
}

var (
	headRe      = regexp.MustCompile(`^(\*+)\s+(.*)`)
	todoRe      = regexp.MustCompile(`\bTODO\b`)
	deadlineRe  = regexp.MustCompile(`DEADLINE:\s*<([^>]+)>`)
	scheduledRe = regexp.MustCompile(`SCHEDULED:\s*<([^>]+)>`)
	tagsRe      = regexp.MustCompile(`\s+(:\S+:)\s*$`)
)

func parseTodos(path string) ([]Todo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var todos []Todo
	var pending *Todo
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if m := headRe.FindStringSubmatch(line); m != nil {
			if pending != nil {
				todos = append(todos, *pending)
				pending = nil
			}
			if todoRe.MatchString(m[2]) {
				title := m[2]
				deadline, scheduled := "", ""
				if d := deadlineRe.FindStringSubmatch(title); d != nil {
					deadline = d[1]
					title = deadlineRe.ReplaceAllString(title, "")
				}
				if s := scheduledRe.FindStringSubmatch(title); s != nil {
					scheduled = s[1]
					title = scheduledRe.ReplaceAllString(title, "")
				}
				title = strings.TrimPrefix(title, "TODO ")
				tags := ""
				if tg := tagsRe.FindStringSubmatch(title); tg != nil {
					raw := strings.Trim(tg[1], ":")
					tags = strings.ReplaceAll(raw, ":", ", ")
					title = strings.TrimSpace(tagsRe.ReplaceAllString(title, ""))
				}
				pending = &Todo{
					Title:     strings.TrimSpace(title),
					File:      path,
					Level:     len(m[1]),
					Deadline:  deadline,
					Scheduled: scheduled,
					Tags:      tags,
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
		}
	}
	if pending != nil {
		todos = append(todos, *pending)
	}
	return todos, scanner.Err()
}

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

func main() {
	cfgPath := defaultConfigPath()
	cfg, err := loadOrCreateConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %s\n", err)
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) > 0 && args[0] == "add" {
		handleAdd(cfg, args[1:])
		return
	}

	handleList(cfg)
}

func handleAdd(cfg *Config, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: org-cli add <title> [due:YYYY-MM-DD] [sched:YYYY-MM-DD] [tags:tag1,tag2]")
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

func handleList(cfg *Config) {
	files, err := findOrgFiles(cfg.Paths)
	if err != nil || len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no org files found")
		return
	}

	var allTodos []Todo
	for _, f := range files {
		todos, _ := parseTodos(f)
		allTodos = append(allTodos, todos...)
	}

	titleW, tagsW, schedW, deadW := len("Title"), len("Tags"), len("Scheduled"), len("Deadline")
	for _, t := range allTodos {
		if len(t.Title) > titleW {
			titleW = len(t.Title)
		}
		if len(t.Tags) > tagsW {
			tagsW = len(t.Tags)
		}
		if len(t.Scheduled) > schedW {
			schedW = len(t.Scheduled)
		}
		if len(t.Deadline) > deadW {
			deadW = len(t.Deadline)
		}
	}

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
