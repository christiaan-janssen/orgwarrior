package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Config struct {
	Paths []string `json:"paths"`
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
		return &cfg, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	cfg := &Config{
		Paths: []string{"~/org/"},
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
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %s\n", p, err)
			continue
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		err = filepath.WalkDir(p, func(fpath string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(fpath, ".org") {
				files = append(files, fpath)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: error walking %s: %s\n", p, err)
		}
	}
	return files, nil
}

type Todo struct {
	Title     string
	File      string
	Level     int
	Deadline  string
	Scheduled string
}

var (
	headRe      = regexp.MustCompile(`^(\*+)\s+(.*)`)
	todoRe      = regexp.MustCompile(`\bTODO\b`)
	deadlineRe  = regexp.MustCompile(`DEADLINE:\s*<([^>]+)>`)
	scheduledRe = regexp.MustCompile(`SCHEDULED:\s*<([^>]+)>`)
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
				deadline := ""
				scheduled := ""
				if d := deadlineRe.FindStringSubmatch(title); d != nil {
					deadline = d[1]
					title = deadlineRe.ReplaceAllString(title, "")
				}
				if s := scheduledRe.FindStringSubmatch(title); s != nil {
					scheduled = s[1]
					title = scheduledRe.ReplaceAllString(title, "")
				}
				title = strings.TrimPrefix(title, "TODO ")
				title = strings.TrimSpace(title)
				pending = &Todo{
					Title:     title,
					File:      path,
					Level:     len(m[1]),
					Deadline:  deadline,
					Scheduled: scheduled,
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

func main() {
	path := defaultConfigPath()
	if path == "" {
		fmt.Fprintln(os.Stderr, "could not determine home directory")
		os.Exit(1)
	}

	cfg, err := loadOrCreateConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %s\n", err)
		os.Exit(1)
	}

	files, err := findOrgFiles(cfg.Paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no org files found")
		os.Exit(1)
	}

	var allTodos []Todo
	for _, f := range files {
		todos, err := parseTodos(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %s\n", f, err)
			continue
		}
		allTodos = append(allTodos, todos...)
	}

	titleW := len("Title")
	schedW := len("Scheduled")
	deadW := len("Deadline")
	for _, t := range allTodos {
		if len(t.Title) > titleW {
			titleW = len(t.Title)
		}
		if len(t.Scheduled) > schedW {
			schedW = len(t.Scheduled)
		}
		if len(t.Deadline) > deadW {
			deadW = len(t.Deadline)
		}
	}

	pad := strings.Repeat(" ", 3)
	fmt.Printf("%-*s%s%-*s%s%s\n", titleW, "Title", pad, schedW, "Scheduled", pad, "Deadline")
	fmt.Println(strings.Repeat("-", titleW+schedW+deadW+len(pad)*2))

	for _, t := range allTodos {
		fmt.Printf("%-*s%s%-*s%s%s\n", titleW, t.Title, pad, schedW, t.Scheduled, pad, t.Deadline)
	}
}
