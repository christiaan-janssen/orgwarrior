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
	if len(args) > 0 && args[0] == "modify" {
		handleModify(cfg, args[1:])
		return
	}
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		printHelp()
		return
	}

	handleList(cfg)
}

func printHelp() {
	fmt.Println(`Usage: orgwarrior [command] [args]

Commands:
  list (default)    List all TODO items
  add <title>       Add a new task
    due:YYYY-MM-DD    Set a deadline
    sched:YYYY-MM-DD  Set a scheduled date
    tags:tag1,tag2    Add tags

  done <id>         Mark a task as done (use the ID from the list)
  modify <id>       Change an existing task's title, dates, or tags
    due:YYYY-MM-DD    Update the deadline
    sched:YYYY-MM-DD  Update the scheduled date
    tags:tag1,tag2    Update the tags

Examples:
  orgwarrior
  orgwarrior add "Fix the bug"
  orgwarrior add "Write docs" due:2026-06-01 sched:2026-05-30 tags:work,docs
  orgwarrior done 3`)
}
