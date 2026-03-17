package ui

import "testing"

func TestIsFullyStaged(t *testing.T) {
	tests := []struct {
		xy   string
		want bool
	}{
		{"M ", true},
		{"A ", true},
		{"D ", true},
		{"R ", true},
		{" M", false},
		{"MM", false},
		{"??", false},
		{"", false},
		{" D", false},
	}
	for _, tt := range tests {
		t.Run(tt.xy, func(t *testing.T) {
			if got := isFullyStaged(tt.xy); got != tt.want {
				t.Errorf("isFullyStaged(%q) = %v, want %v", tt.xy, got, tt.want)
			}
		})
	}
}

func TestGitStatusModel_Navigation(t *testing.T) {
	makeModel := func() GitStatusModel {
		m := NewGitStatusModel()
		m.SetDimensions(80, 20)
		m.entries = []GitStatusEntry{
			{XY: " M", Path: "file1"},
			{XY: "M ", Path: "file2"},
			{XY: "??", Path: "file3"},
		}
		return m
	}

	t.Run("MoveDown", func(t *testing.T) {
		m := makeModel()
		m.MoveDown()
		if m.cursor != 1 {
			t.Errorf("cursor = %d, want 1", m.cursor)
		}
	})

	t.Run("MoveUp", func(t *testing.T) {
		m := makeModel()
		m.cursor = 2
		m.MoveUp()
		if m.cursor != 1 {
			t.Errorf("cursor = %d, want 1", m.cursor)
		}
	})

	t.Run("MoveDown at bottom is no-op", func(t *testing.T) {
		m := makeModel()
		m.cursor = 2
		m.MoveDown()
		if m.cursor != 2 {
			t.Errorf("cursor = %d, want 2", m.cursor)
		}
	})

	t.Run("MoveUp at top is no-op", func(t *testing.T) {
		m := makeModel()
		m.MoveUp()
		if m.cursor != 0 {
			t.Errorf("cursor = %d, want 0", m.cursor)
		}
	})

	t.Run("GoToTop", func(t *testing.T) {
		m := makeModel()
		m.cursor = 2
		m.GoToTop()
		if m.cursor != 0 {
			t.Errorf("cursor = %d, want 0", m.cursor)
		}
	})

	t.Run("GoToBottom", func(t *testing.T) {
		m := makeModel()
		m.GoToBottom()
		if m.cursor != 2 {
			t.Errorf("cursor = %d, want 2", m.cursor)
		}
	})

	t.Run("empty list navigation", func(t *testing.T) {
		m := NewGitStatusModel()
		m.MoveDown()
		m.MoveUp()
		m.GoToTop()
		m.GoToBottom()
		// Should not panic
	})
}

func TestGitStatusModel_Queries(t *testing.T) {
	t.Run("SelectedPath empty", func(t *testing.T) {
		m := NewGitStatusModel()
		if got := m.SelectedPath(); got != "" {
			t.Errorf("SelectedPath() = %q, want empty", got)
		}
	})

	t.Run("SelectedPath", func(t *testing.T) {
		m := NewGitStatusModel()
		m.entries = []GitStatusEntry{{XY: " M", Path: "file1"}}
		if got := m.SelectedPath(); got != "file1" {
			t.Errorf("SelectedPath() = %q, want file1", got)
		}
	})

	t.Run("SelectedEntry empty", func(t *testing.T) {
		m := NewGitStatusModel()
		_, ok := m.SelectedEntry()
		if ok {
			t.Error("SelectedEntry() ok = true, want false")
		}
	})

	t.Run("SelectedEntry", func(t *testing.T) {
		m := NewGitStatusModel()
		m.entries = []GitStatusEntry{{XY: "M ", Path: "file1"}}
		entry, ok := m.SelectedEntry()
		if !ok {
			t.Fatal("SelectedEntry() ok = false, want true")
		}
		if entry.Path != "file1" {
			t.Errorf("entry.Path = %q, want file1", entry.Path)
		}
	})

	t.Run("EntryCount", func(t *testing.T) {
		m := NewGitStatusModel()
		m.entries = []GitStatusEntry{{}, {}, {}}
		if got := m.EntryCount(); got != 3 {
			t.Errorf("EntryCount() = %d, want 3", got)
		}
	})
}

func TestGitStatusModel_SetEntries(t *testing.T) {
	t.Run("clamps cursor when list shrinks", func(t *testing.T) {
		m := NewGitStatusModel()
		m.SetDimensions(80, 20)
		m.entries = make([]GitStatusEntry, 10)
		m.cursor = 8

		m.SetEntries([]GitStatusEntry{{XY: " M", Path: "file1"}})
		if m.cursor != 0 {
			t.Errorf("cursor = %d, want 0", m.cursor)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		m := NewGitStatusModel()
		m.cursor = 5
		m.SetEntries(nil)
		if m.cursor != 0 {
			t.Errorf("cursor = %d, want 0", m.cursor)
		}
	})
}
