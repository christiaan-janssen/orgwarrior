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
	// DateFormat controls how dates are parsed from user input and displayed.
	// Supported values: YYYY-MM-DD (default), DD-MM-YYYY, MM-DD-YYYY.
	DateFormat string `json:"date_format"`
}

// defaultConfigPath returns the standard config file location (~/.config/orgwarrior/config.json).
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "orgwarrior", "config.json")
}

// oldConfigPath returns the legacy config location (~/.config/org-cli/config.json).
func oldConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "org-cli", "config.json")
}

// loadOrCreateConfig reads the config from the given path. If the file doesn't
// exist, it tries the old location (~/.config/org-cli/config.json) and migrates
// it to the new path. If neither exists, it creates a new config at path with
// default values.
func loadOrCreateConfig(path string) (*Config, error) {
	cfg, err := loadConfig(path)
	if err == nil {
		return cfg, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	// Try migrating from old location
	old := oldConfigPath()
	if old != "" && old != path {
		if cfg, err := loadConfig(old); err == nil {
			if e := writeConfig(path, cfg); e != nil {
				return nil, e
			}
			return cfg, nil
		}
	}

	// Create default config
	cfg = defaultConfig()
	if err := writeConfig(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// loadConfig reads and decodes a JSON config file, applying defaults for missing fields.
func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
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
	if cfg.DoneLookbackDays == 0 {
		cfg.DoneLookbackDays = 7
	}
	if cfg.DateFormat == "" {
		cfg.DateFormat = "YYYY-MM-DD"
	}
	return &cfg, nil
}

// defaultConfig returns a Config with sensible defaults.
func defaultConfig() *Config {
	return &Config{
		Paths:            []string{"~/org/"},
		DefaultFile:      "~/org/inbox.org",
		Files:            []string{"inbox.org", "agenda.org"},
		DoneLookbackDays: 7,
		DateFormat:       "YYYY-MM-DD",
	}
}

// writeConfig encodes cfg as JSON and writes it to path, creating directories as needed.
func writeConfig(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
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
