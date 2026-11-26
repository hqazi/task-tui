// Package store persists a model.Store as a human-readable JSON file.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hqazi/task-tui/internal/model"
)

// DefaultPath returns $XDG_DATA_HOME/task-tui/tasks.json, falling back to
// ~/.local/share/task-tui/tasks.json.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "task-tui", "tasks.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "task-tui", "tasks.json"), nil
}

// Load reads the store at path. A missing file yields a new store with a
// default "Inbox" list (nothing is written until Save).
func Load(path string) (*model.Store, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return model.NewStore(), nil
	}
	if err != nil {
		return nil, err
	}
	var s model.Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if s.Version > model.CurrentVersion {
		return nil, fmt.Errorf("%s: unsupported version %d (this build supports up to %d)", path, s.Version, model.CurrentVersion)
	}
	if s.Version == 0 {
		s.Version = model.CurrentVersion
	}
	if s.Lists == nil {
		s.Lists = []*model.List{}
	}
	s.NormalizeFocus()
	return &s, nil
}

// Save writes s to path atomically: it writes a temp file in the same
// directory, syncs it, then renames it over the target.
func Save(path string, s *model.Store) error {
	if s.Version == 0 {
		s.Version = model.CurrentVersion
	}
	if s.FocusOrder == nil {
		s.FocusOrder = []string{}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tasks-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		return fail(err)
	}
	if err := tmp.Sync(); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}
