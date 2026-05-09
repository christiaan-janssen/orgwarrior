# orgwarrior

A minimal CLI for working with Emacs Org-mode files, inspired by Taskwarrior.

After all the hype about vibecoding, I thought to give it a try. I love Taskwarrior, but using org to manage my notes I always wanted an cli to manage my todo's.

Lists TODO items across your org files and adds new ones with scheduled dates, deadlines, and tags.

## Features

- **list** — scan org files and display all TODO items in a table
- **add** — append a new TODO task with `due:`, `sched:`, and `tags:`
- File grouping with per-file headers
- Parses `DEADLINE`, `SCHEDULED` (inline and on following lines)
- Parses org tags (`:tag1:tag2:`)
- Config file at `~/.config/orgwarrior/config.json` (auto-created on first run)

## Install

```bash
go build -o orgwarrior .
```

## Usage

```
orgwarrior                              # list todos
orgwarrior add "Fix the bug"            # add a task
orgwarrior add "Write docs" due:2026-06-01 sched:2026-05-30 tags:work,docs
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
  "files": ["inbox.org", "agenda.org"]
}
```

| Field          | Description                                        |
| -------------- | -------------------------------------------------- |
| `paths`        | Directories or files to scan for .org files        |
| `default_file` | Where `add` appends new tasks                      |
| `files`        | Which .org basenames to display (empty = show all) |
