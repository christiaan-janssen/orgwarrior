// orgwarrior is a simple CLI for working with Emacs Org-mode files.
// It can list TODO items across org files and add new ones.
package main

import (
	"fmt"
	"os"
)

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
	if len(args) > 0 && args[0] == "done" {
		handleDone(cfg, args[1:])
		return
	}

	handleList(cfg)
}
