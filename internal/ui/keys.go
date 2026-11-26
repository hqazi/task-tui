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
}

func newKeyMap() keyMap {
	return keyMap{
		Up:          key.NewBinding(key.WithKeys("up"), key.WithHelp("↑/↓", "move")),
		Down:        key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
		Left:        key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "collapse/parent")),
		Right:       key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "expand")),
		Tab:         key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
		CollapseAll: key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "collapse all")),
		ExpandAll:   key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "expand all")),
		Select:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "toggle section")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
