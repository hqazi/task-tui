# task-tui — Implementation Tasks

The spec is in `REQUIREMENTS.md`. Tasks run in order, and each one is merged to `main` when it's done.
Every task must leave `go build ./...`, `go vet ./...` and `go test ./...` green.

## Architecture (fixed for all tasks)
```
cmd/task-tui/main.go      entry point and --file flag
internal/model/           pure domain: types plus all tree/list operations (no TUI imports)
internal/store/           JSON load/save (atomic write), XDG path
internal/ui/              Bubble Tea app, split by feature:
  app.go      root model, Update dispatch, pane focus, autosave hook
  theme.go    Lip Gloss styles and icons (with ASCII fallback)
  keys.go     key bindings (bubbles/key)
  sidebar.go  list sidebar
  tasks.go    task pane rendering and actions
  focus.go    Focus view
  picker.go   move picker popup
  search.go   / filter
  help.go     hint bar and ? overlay
  confirm.go  y/n modal
  input.go    inline text input (add/edit/rename)
```
UI code calls `internal/model` operations, then saves through `store`. Model operations are unit-tested. UI logic is tested by driving `Update` with `tea.KeyMsg` and asserting on the resulting state.

## T1 — Domain model and storage ✅
- `go mod init github.com/hqazi/task-tui`, add the Charm dependencies, `cmd/task-tui/main.go` stub.
- Types: `Store{Version, Lists, FocusOrder}`, `List{ID, Name, Hidden, Tasks}`, `Task{ID, Text, Done, DoneAt, Today, Collapsed, Children}`.
- Operations (all addressed by ID):
  - Lists: add, rename, delete, move up/down, toggle hidden.
  - Tasks: add after, add at end, add child, edit text, delete subtree, move up/down among siblings, indent, outdent, toggle collapsed, set collapsed on all, toggle done (cascades to descendants, sets `DoneAt`), clear done, move to another list (appended), toggle today.
  - Lookup: find a task and its parent path, and get a breadcrumb.
  - `Flatten(list, hideDone)` gives visible rows with depth, honouring collapsed tasks.
  - Focus: `FocusTasks(now)` returns starred tasks in `FocusOrder`, excluding tasks done before today. Starring appends to the order, and unstarring or deleting removes from it. Also move up/down in focus order.
  - Progress: `done/total` counts over direct children.
- `store`: Load (a missing file gives an empty store with a default "Inbox" list), Save (atomic, via temp file and rename), DefaultPath (XDG).
- Thorough table tests.

## T2 — TUI shell ✅
- Root model with the sidebar (Focus pinned, lists with open counts, a collapsible "Hidden (n)" section) and the task pane (tree with `▾/▸`, `○/✓/★`, progress, dimmed and struck-through done tasks).
- Theme: rounded borders, Catppuccin-like palette, accented active pane, highlighted selection.
- Navigation with `↑/↓`, `Tab` to switch panes, `←/→` to collapse/expand (`←` on a child jumps to its parent), `-`/`+` to collapse/expand all, and `q`/`Ctrl+C` to quit.
- Hint bar and `?` help overlay. Handles terminal resize, truncates long text, and the task pane scrolls.
- `main.go` loads the store (`--file`), runs the program, and autosaves after every mutation.

## T3 — Task actions ✅
- `a` adds below, `A` adds at the end, `s` adds a subtask; `e`/`Enter` edits (inline input, `Esc` cancels); `d` (or `Space`) toggles done; `D` deletes (confirms when the task has children).
- `Shift+↑/↓` reorders, `Tab`/`Shift+Tab` indent/outdent (`Tab` switches panes only when not on a task row; if that's ambiguous, `Ctrl+←/→` can be an alternative), `f` toggles today, `h` hides/shows done, `C` clears done (with confirm).
- The cursor follows the affected task after each operation.

## T4 — List management ✅
- In the sidebar: `n` adds a list, `r` renames, `D` deletes (always confirms), `Shift+↑/↓` reorders, `v` hides/shows, `Enter`/`→` opens the list in the task pane.

## T5 — Focus view ✅
- Selecting Focus shows starred tasks with breadcrumbs (`text · List › Parent`), and a header with the date and done/total.
- `Shift+↑/↓` reorders the focus order; `d`/`Space`, `e`, `f` and `D` act on the real task; `Enter`/`g` jumps to the task in its home list (expanding its ancestors).

## T6 — Move picker and search ✅
- `m` opens a popup of lists (excluding the current one), `↑/↓` + `Enter` moves the task with its subtree, and `Esc` cancels. It works from Focus too.
- `/` opens a fuzzy filter over the current list. `Tab` toggles to all lists, `Enter` jumps to the match, and `Esc` clears.

## T7 — Polish and docs ✅
- `README.md` with screenshot-style ASCII, install (`go install`), and a keybinding table.
- ASCII icon fallback (`TASK_TUI_ASCII=1`), empty-state messages, and a final pass on the look.
