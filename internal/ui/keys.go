package ui

import "github.com/charmbracelet/bubbles/key"

// keyMap is the central keymap. Later tasks add bindings here, then handle
// them in the relevant pane/mode handler and list them in helpGroups.
type keyMap struct {
	Up, Down, Left, Right key.Binding
	Tab                   key.Binding
	CollapseAll           key.Binding
	ExpandAll             key.Binding
	Select                key.Binding
	Help                  key.Binding
	Quit                  key.Binding

	// Task actions (task pane).
	Add, AddEnd, AddChild key.Binding
	Edit, Toggle, Delete  key.Binding
	MoveUp, MoveDown      key.Binding
	Indent, Outdent       key.Binding
	Today, HideDone       key.Binding
	ClearDone, Back       key.Binding

	// Focus view.
	Jump, Unstar, FocusEdit key.Binding

	// List management (sidebar).
	ListAdd, ListRename, ListDelete key.Binding
	ListUp, ListDown, ListHide      key.Binding

	// Shown in the hint bar while an input or confirm is open.
	Submit, Cancel, Yes, No key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:          key.NewBinding(key.WithKeys("up"), key.WithHelp("↑/↓", "move")),
		Down:        key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
		Left:        key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "collapse/parent")),
		Right:       key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "expand")),
		Tab:         key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "open tasks")),
		CollapseAll: key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "collapse all")),
		ExpandAll:   key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "expand all")),
		Select:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "toggle section")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),

		Add:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add below")),
		AddEnd:    key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "add at end")),
		AddChild:  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "subtask")),
		Edit:      key.NewBinding(key.WithKeys("e", "enter"), key.WithHelp("e/enter", "edit")),
		Toggle:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "done")),
		Delete:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		MoveUp:    key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("shift+↑", "move up")),
		MoveDown:  key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("shift+↓", "move down")),
		Indent:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "indent")),
		Outdent:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "outdent")),
		Today:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "today")),
		HideDone:  key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "hide done")),
		ClearDone: key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "clear done")),
		Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "sidebar")),

		Jump:      key.NewBinding(key.WithKeys("enter", "g"), key.WithHelp("enter/g", "jump to list")),
		Unstar:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "unstar")),
		FocusEdit: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),

		ListAdd:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new list")),
		ListRename: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename list")),
		ListDelete: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete list")),
		ListUp:     key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("shift+↑", "list up")),
		ListDown:   key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("shift+↓", "list down")),
		ListHide:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "hide/show list")),

		Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
		Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Yes:    key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes")),
		No:     key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n", "no")),
	}
}
