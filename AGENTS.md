# orgwarrior — AGENTS.md

## Overview
A minimal CLI for working with Emacs Org-mode files. Lists TODO items across org files and adds new ones.

## Build & run
```bash
go build -o orgwarrior .
go vet ./...
./orgwarrior        # list todos
./orgwarrior add "task title" due:YYYY-MM-DD sched:YYYY-MM-DD tags:tag1,tag2
```

## Project structure
```
main.go       — entry point, dispatches to add or list
config.go     — Config struct, ~/.config/orgwarrior/config.json loading/creation
org.go        — Todo struct, org file parsing (regex-based), date formatting
commands.go   — handleAdd, handleList, table output
ROADMAP.md    — features to add
```

## Key conventions
- **Go**: standard library only, no external deps, `package main`
- **Error handling**: print to stderr, `os.Exit(1)` on fatal errors, `continue` on file-level warnings
- **Comments**: doc comments (`//`) on all exported functions/types
- **README**: keep README.md in sync — update features and usage for every new command or option
- **Org parsing**: regex-based; parses `DEADLINE`, `SCHEDULED` inline and on following lines; tags from trailing `:tag:` groups
- **Config**: auto-created at `~/.config/orgwarrior/config.json` on first run with `paths: ["~/org/"]`, `default_file: "~/org/inbox.org"`, and `files: ["inbox.org", "agenda.org"]`
- **Output**: aligned table grouped by file, columns: Title, Tags, Scheduled, Deadline
- **Add command**: appends `* TODO title :tag1:tag2:` followed by `SCHEDULED:`/`DEADLINE:` lines to `default_file`
- **File filtering**: config's `files` array controls which .org files to display (matched by basename)

## Defaults
- Config path: `~/.config/orgwarrior/config.json`
- Org root: `~/org/`
- Inbox file: `~/org/inbox.org`
