// Command task-tui is a terminal task manager.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hqazi/task-tui/internal/store"
	"github.com/hqazi/task-tui/internal/ui"
)

func main() {
	file := flag.String("file", "", "path to the tasks JSON file (default: $XDG_DATA_HOME/task-tui/tasks.json)")
	flag.Parse()

	path := *file
	if path == "" {
		p, err := store.DefaultPath()
		if err != nil {
			fatal(err)
		}
		path = p
	}
	s, err := store.Load(path)
	if err != nil {
		fatal(err)
	}
	if _, err := tea.NewProgram(ui.New(s, path, store.Save), tea.WithAltScreen()).Run(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "task-tui:", err)
	os.Exit(1)
}
