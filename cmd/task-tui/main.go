// Command task-tui is a terminal task manager. The UI arrives in a later task;
// for now main loads the store and prints a summary.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hqazi/task-tui/internal/store"
)

func main() {
	file := flag.String("file", "", "path to the tasks JSON file (default: $XDG_DATA_HOME/task-tui/tasks.json)")
	flag.Parse()

	path := *file
	if path == "" {
		p, err := store.DefaultPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "task-tui:", err)
			os.Exit(1)
		}
		path = p
	}
	s, err := store.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "task-tui:", err)
		os.Exit(1)
	}
	fmt.Printf("%d lists\n", len(s.Lists))
}
