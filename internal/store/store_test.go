package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hqazi/task-tui/internal/model"
)

func sample(t *testing.T) *model.Store {
	t.Helper()
	s := model.NewStore()
	w, err := s.AddList("Work")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.AddTaskEnd(w.ID, "Ship v2")
	b, _ := s.AddChild(a.ID, "Write changelog")
	c, _ := s.AddChild(a.ID, "Tag release")
	_ = s.ToggleToday(b.ID)
	_ = s.ToggleToday(a.ID)
	_ = s.ToggleDone(c.ID, time.Date(2026, 11, 26, 9, 30, 0, 0, time.UTC))
	_ = s.ToggleCollapsed(a.ID)
	_ = s.ToggleListHidden(s.Lists[0].ID)
	return s
}

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "dir", "tasks.json")
	want := sample(t)
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		a, _ := json.MarshalIndent(want, "", " ")
		b, _ := json.MarshalIndent(got, "", " ")
		t.Fatalf("round trip mismatch\nwant %s\ngot  %s", a, b)
	}
}

func TestMissingFileGivesDefaultInbox(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope", "tasks.json")
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Lists) != 1 || s.Lists[0].Name != "Inbox" || s.Version != model.CurrentVersion {
		t.Fatalf("got %+v", s)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("Load must not create the file")
	}
}

func TestSaveIsAtomicAndLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")
	s := sample(t)
	for i := 0; i < 3; i++ {
		if err := Save(path, s); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "tasks.json" {
		t.Fatalf("unexpected files: %v", entries)
	}
}

func TestSaveFailureKeepsOldFileIntact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")
	if err := Save(path, sample(t)); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	// A directory squatting on the target makes rename fail.
	bad := filepath.Join(dir, "blocked")
	if err := os.Mkdir(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Save(bad, model.NewStore()); err == nil {
		t.Fatal("expected error")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("existing file changed")
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestFileFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	if err := Save(path, sample(t)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	text := string(data)
	for _, want := range []string{`"version": 1`, `"focusOrder"`, `"doneAt": null`, "\n  "} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name, content, wantErr string
	}{
		{"garbage", "not json", "parse"},
		{"empty", "", "parse"},
		{"future version", `{"version": 99, "lists": []}`, "unsupported version"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tasks.json")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got %v want contains %q", err, tc.wantErr)
			}
		})
	}
}

func TestLoadNormalizesFocusAndDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	content := `{"lists":[{"id":"l1","name":"W","hidden":false,"tasks":[
	  {"id":"a","text":"A","done":false,"doneAt":null,"today":true,"collapsed":false},
	  {"id":"b","text":"B","done":false,"doneAt":null,"today":true,"collapsed":false}]}],
	  "focusOrder":["ghost","b"]}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Version != model.CurrentVersion {
		t.Errorf("version %d", s.Version)
	}
	if got := strings.Join(s.FocusOrder, ","); got != "b,a" {
		t.Errorf("focus %s", got)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Run("xdg", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/xdg/data")
		got, err := DefaultPath()
		if err != nil || got != filepath.Join("/xdg/data", "task-tui", "tasks.json") {
			t.Fatalf("got %q %v", got, err)
		}
	})
	t.Run("fallback", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", "/home/someone")
		got, err := DefaultPath()
		want := filepath.Join("/home/someone", ".local", "share", "task-tui", "tasks.json")
		if err != nil || got != want {
			t.Fatalf("got %q %v want %q", got, err, want)
		}
	})
}
