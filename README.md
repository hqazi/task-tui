# task-tui

A good-looking terminal task manager built around lists, nested tasks and a daily **Focus** view. Tasks are plain strings that can be nested to any depth, completed, collapsed and starred for today. Everything autosaves to a single human-readable JSON file, and the app ships as one binary built with Bubble Tea, Lip Gloss and Bubbles.

```
╭─────────────────╮╭─────────────────────────────────────────────────────────╮
│ Lists           ││ Work                                                    │
│ ★ Focus       2 ││ ▾ ○ Ship v2                                         1/2 │
│   Work        9 ││   ▾ ○ Write changelog                             0/1 ★ │
│   Home        1 ││       ○ Collect merged PR titles from the last three s… │
│   Ideas       1 ││     ✓ Tag release                                     ★ │
│ ▸ Hidden (1)    ││ ▾ ○ Refactor auth                                   0/3 │
│                 ││     ○ Extract token store                               │
│                 ││     ○ Drop legacy sessions                              │
│                 ││     ○ Add tests                                         │
│                 ││   ○ Reply to Sam                                      ★ │
╰─────────────────╯╰─────────────────────────────────────────────────────────╯
 a add below · s subtask · e/enter edit · space done · t today · d delete · t…
```

## Install

```sh
go install github.com/hqazi/task-tui/cmd/task-tui@latest
```

Requires Go 1.24 or newer.

## Usage

```sh
task-tui [--file path]
```

Data lives in `$XDG_DATA_HOME/task-tui/tasks.json`, falling back to `~/.local/share/task-tui/tasks.json`. Use `--file` to point at a different file. A missing file starts you with an empty store and one list called "Inbox". Every change is saved immediately (atomic write via temp file and rename).

Set `TASK_TUI_ASCII=1` for terminals without good Unicode support. Icons become `[ ]`, `[x]`, `*`, `>`, `v` and borders become plain lines.

## Focus

Focus is a virtual list pinned at the top of the sidebar. It gathers every task you starred with `t`, from any list and at any depth (including hidden lists), and shows each one with a breadcrumb:

```
╭─────────────────╮╭─────────────────────────────────────────────────────────╮
│ Lists           ││ FOCUS · Thu 26 Nov · 2/4                                │
│ ★ Focus       2 ││ ○ Write changelog · Work › Ship v2                      │
│   Work        9 ││ ○ Reply to Sam · Work                                   │
│   Home        1 ││ ✓ Buy groceries · Home                                  │
│   Ideas       1 ││ ✓ Tag release · Work › Ship v2                          │
╰─────────────────╯╰─────────────────────────────────────────────────────────╯
 ↑/↓ move · shift+↑ move up · space done · e edit · t unstar · d delete · …
```

- Focus has its **own manual order** (`shift+↑/↓`), separate from the order in each list. Newly starred tasks go to the bottom.
- Completing, editing, unstarring, moving or deleting a task in Focus acts on the real task. `enter` or `g` jumps to the task in its home list.
- Stars never clear on their own. A task leaves Focus when you unstar it, or the day after you completed it. Done tasks stay visible, struck through, for the rest of that day.
- The header shows today's date and progress, for example `FOCUS · Thu 26 Nov · 2/5`.

## Keybindings

Press `?` in the app for the same list.

### Sidebar

| Key | Action |
|---|---|
| `↑/↓` | move |
| `tab` | open tasks (`→` and `enter` also work on a list) |
| `n` | new list |
| `r` | rename list |
| `d` | delete list (always asks) |
| `shift+↑` / `shift+↓` | list up / list down |
| `v` | hide/show list |
| `enter` | toggle section (on "Hidden (n)") |
| `?` | help |
| `q` | quit |

### Tasks

| Key | Action |
|---|---|
| `↑/↓` | move |
| `esc` | back to sidebar |
| `a` | add below |
| `A` | add at end |
| `s` | subtask |
| `e` / `enter` | edit |
| `space` | done (completes all subtasks too) |
| `d` | delete (asks if it has subtasks) |
| `t` | today (star) |
| `h` | hide done |
| `C` | clear done (asks) |
| `m` | move to list |
| `/` | search |
| `shift+↑` / `shift+↓` | move up / move down |
| `tab` | indent |
| `shift+tab` | outdent |
| `←` | collapse/parent |
| `→` | expand |
| `-` | collapse all |
| `+` | expand all |
| `?` | help |
| `q` | quit |

### Focus

| Key | Action |
|---|---|
| `↑/↓` | move |
| `enter` / `g` | jump to list |
| `t` | unstar |
| `shift+↑` / `shift+↓` | move up / move down |
| `space` | done |
| `e` | edit |
| `d` | delete |
| `m` | move to list |
| `/` | search |
| `esc` | back to sidebar |
| `?` | help |
| `q` | quit |

### Popups

| Popup | Key | Action |
|---|---|---|
| Text input | `enter` / `esc` | save / cancel |
| Confirm | `y` / `n` | yes / no (`esc` also cancels) |
| Move picker | `↑/↓`, `enter`, `esc` | choose, move, cancel |
| Search | type, `↑/↓`, `tab`, `enter`, `esc` | filter, select, switch this list / all lists, jump, close |
| Help | any key | close |

## License

Apache 2.0, see [LICENSE](LICENSE).
