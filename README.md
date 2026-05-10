# orgwarrior

A minimal CLI for working with Emacs Org-mode files, inspired by Taskwarrior.

After all the hype about vibecoding, I thought to give it a try. I love Taskwarrior, but using org to manage my notes I always wanted a CLI to manage my todos.

## Features

- **list** — scan org files and display all TODO items in a table with IDs
- **add** — append a new TODO task with `due:`, `sched:`, and `tags:`
- **done** — mark a task complete (TODO → DONE + CLOSED timestamp)
- **modify** — change a task's title, deadline, scheduled date, or tags
- **delete** — remove a task (also `del`)
- **completed** — show tasks completed this week (also `comp`)
- **Filters** — `tag:work`, `due:before:+7d`, `due:after:yesterday`, etc.
- **Relative dates** — `due:fri`, `sched:+2d`, `due:tomorrow`, `sched:before:-1w`
- **Colors** — past dates in red, future dates in green, file headers in cyan (auto-detects terminal)
- **Date format** — configurable `date_format` (YYYY-MM-DD, DD-MM-YYYY, MM-DD-YYYY) for input and display
- File grouping with per-file headers
- Parses `DEADLINE`, `SCHEDULED` (inline and on following lines)
- Parses org tags (`:tag1:tag2:`)
- Config file at `~/.config/orgwarrior/config.json` (auto-created on first run)

## Install

```bash
git clone https://github.com/christiaan-janssen/orgwarrior.git
cd orgwarrior
go build -o orgwarrior .
```

## Usage

```
orgwarrior                                         # list all todos
orgwarrior add "Fix the bug"                       # add a task
orgwarrior add "Write docs" due:fri sched:+1d      # relative dates
orgwarrior add "Meeting" due:tomorrow tags:work     # also: today, yesterday, mon..sun
orgwarrior done 3                                  # mark task #3 as done
orgwarrior modify 2 "New title"                    # change title
orgwarrior modify 4 due:2026-07-01                 # change deadline only
orgwarrior delete 5                                # remove task #5
orgwarrior completed                              # recently done tasks
orgwarrior tag:work                                # filter by tag
orgwarrior due:before:+7d                          # filter by deadline
orgwarrior sched:after:-1w                         # offsets: +3d, -1w, +2m
```

## Built with AI

- **Client**: [opencode](https://opencode.ai)
- **Model**: Big Pickel
- **Approach**: vibecoding — all code was generated through natural language prompts

## Config

Auto-created at `~/.config/orgwarrior/config.json`:

```json
{
  "paths": ["~/org/"],
  "default_file": "~/org/inbox.org",
  "files": ["inbox.org", "agenda.org"],
  "date_format": "YYYY-MM-DD"
}
```

| Field               | Description                                              |
| ------------------- | -------------------------------------------------------- |
| `paths`             | Directories or files to scan for .org files              |
| `default_file`      | Where `add` appends new tasks                            |
| `files`             | Which .org basenames to display (empty = show all)       |
| `done_lookback_days`| Days to look back for completed tasks (default 7)        |
| `date_format`       | Date input/display format (YYYY-MM-DD, DD-MM-YYYY, MM-DD-YYYY) |
