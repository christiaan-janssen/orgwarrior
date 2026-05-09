package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the user configuration loaded from ~/.config/orgwarrior/config.json.
type Config struct {
	// Paths is a list of directories or files to scan for .org files.
	Paths []string `json:"paths"`
	// DefaultFile is where new tasks are appended by the "add" command.
	DefaultFile string `json:"default_file"`
	// Files is a whitelist of basenames to track (e.g. ["inbox.org", "agenda.org"]).
	// Empty means all found .org files are shown.
	Files []string `json:"files"`
	// DoneLookbackDays controls how many days back to show completed tasks.
	// Default is 7.
	DoneLookbackDays int `json:"done_lookback_days"`
}

// defaultConfigPath returns the standard location for the config file.
// Checks the new name (orgwarrior) first, then falls back to the old name (org-cli).
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	newPath := filepath.Join(home, ".config", "orgwarrior", "config.json")
	if _, err := os.Stat(newPath); err == nil {
		return newPath
	}
	oldPath := filepath.Join(home, ".config", "org-cli", "config.json")
	if _, err := os.Stat(oldPath); err == nil {
		return oldPath
	}
	return newPath
}

// loadOrCreateConfig reads the config from path. If the file doesn't exist,
// it creates it with default values (paths: ["~/org/"], default_file: "~/org/inbox.org").
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
		if len(cfg.Files) == 0 {
			cfg.Files = []string{"inbox.org", "agenda.org"}
		}
		return &cfg, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	cfg := &Config{
		Paths:       []string{"~/org/"},
		DefaultFile: "~/org/inbox.org",
		Files:       []string{"inbox.org", "agenda.org"},
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

// expandPath resolves a leading "~/" to the user's home directory.
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// findOrgFiles walks each path in the list and collects all .org files.
// Directories are walked recursively; individual files are included as-is.
// Paths that don't exist are silently skipped.
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
