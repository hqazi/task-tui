# task-tui — Requirements

A good-looking terminal task manager built around lists, nested tasks, and a daily **Focus** view.

## 1. Stack
- **Go** with **Bubble Tea** (app loop), **Lip Gloss** (styling) and **Bubbles** (text input, viewport).
- Ships as a single binary.

## 2. Core concepts
| Concept | Description |
|---|---|
| **List** | A named collection of tasks, such as "Work" or "Home". It can be hidden. |
| **Task** | A plain string with no title/description split. It can be done or not done, starred for Today or not, and collapsed or expanded. It can have subtasks. |
| **Subtask** | A task nested under another task. Nesting depth is **unlimited**. |
| **Today star (★)** | A flag on any task, at any depth, meaning "I'm doing this today". |
| **Focus view** | A virtual list that gathers every starred task from all lists, in its own order. |

## 3. Layout
```
┌ Lists ─────┐┌ Work ─────────────────────────┐
│ ★ Focus  3 ││ ▾ ○ Ship v2               1/2 │
│ ▸ Work   5 ││     ○ Write changelog       ★ │
│   Home   2 ││     ✓ Tag release             │
│   Ideas  9 ││ ▸ ○ Refactor auth         0/3 │
│            ││   ○ Reply to Sam              │
└────────────┘└───────────────────────────────┘
 a add · s subtask · space done · f focus · m move · ? help
```
- **Left sidebar** has Focus pinned at the top, then the user's lists. Each list shows its count of open tasks.
- **Right pane** shows the tasks of the selected list, or of Focus.
- **Bottom hint bar** shows context-sensitive key hints, and `?` opens a full help overlay.
- **Focus switching:** from the sidebar, `Tab`, `→` or `Enter` moves to the task pane. In the task pane `Tab` / `Shift+Tab` indent / outdent the task (see §5), so `Esc` returns focus to the sidebar.
- The layout adapts to the terminal size, and long task text wraps or truncates cleanly.

## 4. Lists
- **Add** a list (`n` in the sidebar) and **rename** it (`r`).
- **Delete** a list (`d`). This always asks for y/n confirmation.
- **Reorder** lists (`Shift+↑/↓`).
- **Hide/show** a list (`v`). Hidden lists are not deleted. They collapse into a "Hidden (n)" section at the bottom of the sidebar, which can be expanded to reach them.

## 5. Tasks
| Action | Key | Notes |
|---|---|---|
| Navigate | `↑/↓` | |
| Add sibling below cursor | `a` | Opens an inline text input. |
| Add at end of list | `A` | |
| Add subtask | `s` | Adds the new task as the last child of the selected task and expands the parent. |
| Edit text | `e` / `Enter` | Inline edit. `Esc` cancels. |
| Toggle complete | `Space` | |
| Delete | `d` | Deletes immediately for a task with no subtasks, and asks y/n for a task that has subtasks. |
| Reorder | `Shift+↑/↓` | Moves the task, with its subtree, among its siblings. |
| Indent / outdent | `Tab` / `Shift+Tab` | In the task pane only (`Esc` returns to the sidebar). Makes the task a child of the sibling above it, or of its grandparent. |
| Collapse / expand | `←` / `→` | `←` on a child jumps to its parent. |
| Collapse all / expand all | `-` / `+` | |
| Move to another list | `m` | Opens a picker popup of lists. The task moves with its subtree and is appended at the bottom of the target list. |
| Star for Today | `f` | Toggle. |
| Hide/show done tasks | `h` | Per-list toggle (UI state, not saved). |
| Clear done tasks | `C` | Asks y/n, then removes the done tasks in the current list. |
| Search/filter | `/` | Fuzzy filter. `Tab` inside search switches between this list and all lists. `Esc` clears the filter. |

### Completion rules
- A done task stays in place, dimmed and struck through.
- Completing a parent **completes all of its descendants**.
- Completing every child does **not** complete the parent automatically. Parents show progress as `done/total` for their direct children.
- Un-completing a parent does not un-complete its children.

### Collapsed tasks
- A collapsed task shows `▸` and its subtask count. An expanded task shows `▾`.
- The collapse state is saved.

## 6. Focus view (prioritisation)
- Lists every starred task from all lists. Each task shows a breadcrumb, for example `Write changelog · Work › Ship v2`. Only the starred task is shown, not its subtree.
- Focus has **its own manual order** (`Shift+↑/↓`), separate from each task's position in its list. Newly starred tasks are added at the bottom.
- Actions taken in Focus, such as completing, editing, unstarring, or deleting a task, apply to the real task.
- `Enter`/`g` on a task in Focus jumps to it in its home list.
- Stars **never clear automatically**. A task is unstarred manually, or it drops out of Focus **the day after it was completed** (done tasks stay visible and struck through for the rest of the day they were finished).
- The header shows today's date and progress, for example `FOCUS · Thu 26 Nov · 2/5`.

## 7. Storage
- A single JSON file at `$XDG_DATA_HOME/task-tui/tasks.json`, which falls back to `~/.local/share/task-tui/tasks.json`.
- **Autosave** on every change, using an atomic write (write a temp file, then rename).
- The file is human-readable and includes a `version` field for future migrations.
- An optional `--file <path>` flag points the app at a different data file.

Sketch:
```json
{
  "version": 1,
  "lists": [
    { "id": "l1", "name": "Work", "hidden": false,
      "tasks": [
        { "id": "t1", "text": "Ship v2", "done": false, "doneAt": null,
          "today": false, "collapsed": false,
          "children": [ { "id": "t2", "text": "Write changelog", "today": true, "...": "..." } ] }
      ] }
  ],
  "focusOrder": ["t2"]
}
```

## 8. Look & feel
- A single polished default theme with rounded borders and subtle colours, using a Catppuccin-like palette.
- The selected row is highlighted, the active pane has an accented border, and done tasks are dimmed.
- Icons are `○` open, `✓` done, `★` today, `▸/▾` collapsed/expanded, with ASCII fallbacks.

## 9. Out of scope for v1
Undo/redo, multiple themes, mouse support, configurable keybindings, due dates, priority levels, sync, and cut & paste moves.

## 10. Decisions on earlier open questions
1. Starred tasks in **hidden** lists still appear in Focus.
2. `m` (move) also works from Focus, using the same picker.
