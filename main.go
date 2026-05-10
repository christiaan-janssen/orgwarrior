// orgwarrior is a simple CLI for working with Emacs Org-mode files.
// It can list TODO items across org files and add new ones.
package main

import (
	"fmt"
	"os"
	"strings"
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
	if len(args) > 0 && (args[0] == "delete" || args[0] == "del") {
		handleDelete(cfg, args[1:])
		return
	}
	if len(args) > 0 && (args[0] == "completed" || args[0] == "complete" || args[0] == "comp") {
		handleCompleted(cfg)
		return
	}
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		printHelp()
		return
	}
	if len(args) > 0 && args[0] == "list" {
		handleList(cfg, args[1:])
		return
	}

	if len(args) > 0 && !isKnownCommand(args[0]) && !isFilterArg(args[0]) && !startsFlag(args[0]) {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printHelp()
		os.Exit(1)
	}

	handleList(cfg, args)
}

func isKnownCommand(s string) bool {
	switch s {
	case "add", "done", "modify", "delete", "del", "completed", "complete", "comp", "list":
		return true
	}
	return false
}

func isFilterArg(s string) bool {
	return strings.HasPrefix(s, "tag:") || strings.HasPrefix(s, "due:") || strings.HasPrefix(s, "sched:")
}

func startsFlag(s string) bool {
	return strings.HasPrefix(s, "-")
}

func printHelp() {
	fmt.Println(`Usage: orgwarrior [command] [args]

Commands:
  list (default)    List all TODO items
  add <title>       Add a new task
    due:DATE          Set a deadline
    sched:DATE        Set a scheduled date
    tags:tag1,tag2    Add tags

  done <id>         Mark a task as done (use the ID from the list)
  modify <id>       Change an existing task's title, dates, or tags
    due:DATE          Update the deadline
    sched:DATE        Update the scheduled date
    tags:tag1,tag2    Update the tags

  delete <id>       Remove a task (also: del)
  completed         Show recently completed tasks (also: complete, comp)

Configuration (~/.config/orgwarrior/config.json):
  date_format    Date display/input format (YYYY-MM-DD, DD-MM-YYYY, MM-DD-YYYY)
                 Default: YYYY-MM-DD

Dates:
  Absolute:      YYYY-MM-DD (or DD-MM-YYYY / MM-DD-YYYY per config)
  Relative:      today, tomorrow, yesterday, mon/tue/wed/thu/fri/sat/sun
  Offsets:       +3d, +1w, +2m (also -3d, -1w for filters)

Examples:
  orgwarrior
  orgwarrior add "Fix the bug"
  orgwarrior add "Write docs" due:fri sched:+1d tags:work,docs
  orgwarrior done 3
  orgwarrior completed             # show tasks completed this week
  orgwarrior tag:work              # list only tasks tagged "work"
  orgwarrior due:before:+7d        # tasks due within the next week
  orgwarrior sched:after:yesterday # tasks scheduled from today onward`)
}
