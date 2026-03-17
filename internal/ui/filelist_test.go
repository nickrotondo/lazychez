package ui

import (
	"testing"

	"github.com/nickrotondo/lazychez/internal/chezmoi"
)

func TestDriftKindSortOrder(t *testing.T) {
	tests := []struct {
		drift DriftKind
		want  int
	}{
		{DriftDestEdited, 0},
		{DriftSourceEdited, 1},
		{DriftNone, 2},
	}
	for _, tt := range tests {
		if got := tt.drift.sortOrder(); got != tt.want {
			t.Errorf("DriftKind(%d).sortOrder() = %d, want %d", tt.drift, got, tt.want)
		}
	}
}

func TestFileItemHasDrift(t *testing.T) {
	tests := []struct {
		name        string
		item        FileItem
		wantHasDrift bool
	}{
		{"no drift", FileItem{SourceState: ' ', DestState: ' '}, false},
		{"source modified", FileItem{SourceState: 'M', DestState: ' '}, true},
		{"dest modified", FileItem{SourceState: ' ', DestState: 'M'}, true},
		{"both modified", FileItem{SourceState: 'M', DestState: 'M'}, true},
		{"heading always false", FileItem{SourceState: 'M', DestState: ' ', IsHeading: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.HasDrift(); got != tt.wantHasDrift {
				t.Errorf("HasDrift() = %v, want %v", got, tt.wantHasDrift)
			}
		})
	}
}

func TestMergeFilesWithStatus(t *testing.T) {
	t.Run("no status entries", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
			{Path: ".gitconfig", SourceRelPath: "dot_gitconfig"},
		}
		items := MergeFilesWithStatus(managed, nil, nil)
		if len(items) != 2 {
			t.Fatalf("got %d items, want 2", len(items))
		}
		for _, item := range items {
			if item.Drift != DriftNone {
				t.Errorf("item %q has drift %d, want DriftNone", item.Path, item.Drift)
			}
		}
	})

	t.Run("dest edited when not in git modified", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		status := []chezmoi.StatusEntry{
			{SourceState: ' ', DestState: 'M', Path: ".zshrc"},
		}
		items := MergeFilesWithStatus(managed, status, nil)
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
		if items[0].Drift != DriftDestEdited {
			t.Errorf("drift = %d, want DriftDestEdited", items[0].Drift)
		}
		if items[0].DestState != 'M' {
			t.Errorf("DestState = %c, want M", items[0].DestState)
		}
	})

	t.Run("source edited when in git modified", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		status := []chezmoi.StatusEntry{
			{SourceState: 'M', DestState: ' ', Path: ".zshrc"},
		}
		gitModified := map[string]bool{"dot_zshrc": true}
		items := MergeFilesWithStatus(managed, status, gitModified)
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
		if items[0].Drift != DriftSourceEdited {
			t.Errorf("drift = %d, want DriftSourceEdited", items[0].Drift)
		}
	})

	t.Run("orphan status entry appended as dest edited", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		status := []chezmoi.StatusEntry{
			{SourceState: ' ', DestState: 'A', Path: ".bashrc"},
		}
		items := MergeFilesWithStatus(managed, status, nil)
		if len(items) != 2 {
			t.Fatalf("got %d items, want 2", len(items))
		}
		// Find the orphan
		var orphan *FileItem
		for i := range items {
			if items[i].Path == ".bashrc" {
				orphan = &items[i]
			}
		}
		if orphan == nil {
			t.Fatal("orphan .bashrc not found")
		}
		if orphan.Drift != DriftDestEdited {
			t.Errorf("orphan drift = %d, want DriftDestEdited", orphan.Drift)
		}
	})

	t.Run("sort order dest then source then synced", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
			{Path: ".bashrc", SourceRelPath: "dot_bashrc"},
			{Path: ".vimrc", SourceRelPath: "dot_vimrc"},
		}
		status := []chezmoi.StatusEntry{
			{SourceState: 'M', DestState: ' ', Path: ".zshrc"},  // source edited
			{SourceState: ' ', DestState: 'M', Path: ".bashrc"}, // dest edited
		}
		gitModified := map[string]bool{"dot_zshrc": true}
		items := MergeFilesWithStatus(managed, status, gitModified)

		if len(items) != 3 {
			t.Fatalf("got %d items, want 3", len(items))
		}
		// dest edited first, then source edited, then synced
		if items[0].Path != ".bashrc" || items[0].Drift != DriftDestEdited {
			t.Errorf("items[0] = %q drift=%d, want .bashrc DriftDestEdited", items[0].Path, items[0].Drift)
		}
		if items[1].Path != ".zshrc" || items[1].Drift != DriftSourceEdited {
			t.Errorf("items[1] = %q drift=%d, want .zshrc DriftSourceEdited", items[1].Path, items[1].Drift)
		}
		if items[2].Path != ".vimrc" || items[2].Drift != DriftNone {
			t.Errorf("items[2] = %q drift=%d, want .vimrc DriftNone", items[2].Path, items[2].Drift)
		}
	})

	t.Run("alphabetical within same drift group", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
			{Path: ".bashrc", SourceRelPath: "dot_bashrc"},
			{Path: ".config", SourceRelPath: "dot_config"},
		}
		items := MergeFilesWithStatus(managed, nil, nil)
		// All DriftNone, should be sorted alphabetically
		if items[0].Path != ".bashrc" {
			t.Errorf("items[0].Path = %q, want .bashrc", items[0].Path)
		}
		if items[1].Path != ".config" {
			t.Errorf("items[1].Path = %q, want .config", items[1].Path)
		}
		if items[2].Path != ".zshrc" {
			t.Errorf("items[2].Path = %q, want .zshrc", items[2].Path)
		}
	})

	t.Run("empty inputs", func(t *testing.T) {
		items := MergeFilesWithStatus(nil, nil, nil)
		if len(items) != 0 {
			t.Errorf("got %d items, want 0", len(items))
		}
	})
}

func TestInsertHeadings(t *testing.T) {
	t.Run("no drift files produces no headings", func(t *testing.T) {
		files := []FileItem{
			{Path: ".zshrc", SourceState: ' ', DestState: ' '},
			{Path: ".vimrc", SourceState: ' ', DestState: ' '},
		}
		result := insertHeadings(files)
		for _, f := range result {
			if f.IsHeading {
				t.Error("unexpected heading in result with no drift")
			}
		}
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
	})

	t.Run("single drift group gets heading", func(t *testing.T) {
		files := []FileItem{
			{Path: ".zshrc", SourceState: ' ', DestState: 'M', Drift: DriftDestEdited},
			{Path: ".vimrc", SourceState: ' ', DestState: ' ', Drift: DriftNone},
		}
		result := insertHeadings(files)
		// Should have: heading(dest), .zshrc, heading(synced), .vimrc
		if len(result) != 4 {
			t.Fatalf("len = %d, want 4", len(result))
		}
		if !result[0].IsHeading {
			t.Error("result[0] should be heading")
		}
		if result[1].Path != ".zshrc" {
			t.Errorf("result[1].Path = %q, want .zshrc", result[1].Path)
		}
		if !result[2].IsHeading {
			t.Error("result[2] should be heading")
		}
		if result[3].Path != ".vimrc" {
			t.Errorf("result[3].Path = %q, want .vimrc", result[3].Path)
		}
	})

	t.Run("all three groups get headings", func(t *testing.T) {
		files := []FileItem{
			{Path: ".bashrc", Drift: DriftDestEdited, SourceState: ' ', DestState: 'M'},
			{Path: ".zshrc", Drift: DriftSourceEdited, SourceState: 'M', DestState: ' '},
			{Path: ".vimrc", Drift: DriftNone, SourceState: ' ', DestState: ' '},
		}
		result := insertHeadings(files)
		// heading + file for each of 3 groups = 6
		if len(result) != 6 {
			t.Fatalf("len = %d, want 6", len(result))
		}
		headingCount := 0
		for _, f := range result {
			if f.IsHeading {
				headingCount++
			}
		}
		if headingCount != 3 {
			t.Errorf("headingCount = %d, want 3", headingCount)
		}
	})

	t.Run("heading text values", func(t *testing.T) {
		if got := headingText(DriftDestEdited); got != "dest edited · space to add" {
			t.Errorf("headingText(DriftDestEdited) = %q", got)
		}
		if got := headingText(DriftSourceEdited); got != "source edited · a to apply" {
			t.Errorf("headingText(DriftSourceEdited) = %q", got)
		}
		if got := headingText(DriftNone); got != "synced" {
			t.Errorf("headingText(DriftNone) = %q", got)
		}
	})
}

func TestFileListModel_Navigation(t *testing.T) {
	makeModel := func() FileListModel {
		m := NewFileListModel()
		m.SetDimensions(80, 20)
		// Simulate files with a heading: [heading, file0, heading, file1, file2]
		m.files = []FileItem{
			{IsHeading: true, HeadingText: "group1"},
			{Path: ".bashrc"},
			{IsHeading: true, HeadingText: "group2"},
			{Path: ".vimrc"},
			{Path: ".zshrc"},
		}
		m.cursor = 1 // start on first file
		return m
	}

	t.Run("MoveDown skips heading", func(t *testing.T) {
		m := makeModel()
		m.MoveDown() // from .bashrc (1), should skip heading (2), land on .vimrc (3)
		if m.cursor != 3 {
			t.Errorf("cursor = %d, want 3", m.cursor)
		}
	})

	t.Run("MoveUp skips heading", func(t *testing.T) {
		m := makeModel()
		m.cursor = 3 // .vimrc
		m.MoveUp()   // should skip heading (2), land on .bashrc (1)
		if m.cursor != 1 {
			t.Errorf("cursor = %d, want 1", m.cursor)
		}
	})

	t.Run("MoveDown at bottom is no-op", func(t *testing.T) {
		m := makeModel()
		m.cursor = 4 // .zshrc (last)
		m.MoveDown()
		if m.cursor != 4 {
			t.Errorf("cursor = %d, want 4", m.cursor)
		}
	})

	t.Run("MoveUp at top is no-op", func(t *testing.T) {
		m := makeModel()
		m.cursor = 1 // .bashrc (first file)
		m.MoveUp()
		if m.cursor != 1 {
			t.Errorf("cursor = %d, want 1", m.cursor)
		}
	})

	t.Run("GoToTop lands on first file not heading", func(t *testing.T) {
		m := makeModel()
		m.cursor = 4
		m.GoToTop()
		if m.cursor != 1 {
			t.Errorf("cursor = %d, want 1 (first file, not heading at 0)", m.cursor)
		}
	})

	t.Run("GoToBottom lands on last file", func(t *testing.T) {
		m := makeModel()
		m.cursor = 1
		m.GoToBottom()
		if m.cursor != 4 {
			t.Errorf("cursor = %d, want 4", m.cursor)
		}
	})

	t.Run("empty list navigation", func(t *testing.T) {
		m := NewFileListModel()
		m.MoveDown()
		m.MoveUp()
		m.GoToTop()
		m.GoToBottom()
		// Should not panic
	})
}

func TestFileListModel_Queries(t *testing.T) {
	t.Run("SelectedPath empty list", func(t *testing.T) {
		m := NewFileListModel()
		if got := m.SelectedPath(); got != "" {
			t.Errorf("SelectedPath() = %q, want empty", got)
		}
	})

	t.Run("SelectedPath on heading returns empty", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{{IsHeading: true, HeadingText: "test"}}
		m.cursor = 0
		if got := m.SelectedPath(); got != "" {
			t.Errorf("SelectedPath() = %q, want empty", got)
		}
	})

	t.Run("SelectedPath on file", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{{Path: ".zshrc"}}
		m.cursor = 0
		if got := m.SelectedPath(); got != ".zshrc" {
			t.Errorf("SelectedPath() = %q, want .zshrc", got)
		}
	})

	t.Run("SelectedItem nil on heading", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{{IsHeading: true}}
		m.cursor = 0
		if got := m.SelectedItem(); got != nil {
			t.Error("SelectedItem() should be nil for heading")
		}
	})

	t.Run("SelectedItem returns item", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{{Path: ".zshrc", Drift: DriftDestEdited}}
		m.cursor = 0
		got := m.SelectedItem()
		if got == nil {
			t.Fatal("SelectedItem() = nil, want item")
		}
		if got.Path != ".zshrc" {
			t.Errorf("SelectedItem().Path = %q, want .zshrc", got.Path)
		}
	})

	t.Run("FileCount excludes headings", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{
			{IsHeading: true},
			{Path: ".zshrc"},
			{IsHeading: true},
			{Path: ".vimrc"},
		}
		if got := m.FileCount(); got != 2 {
			t.Errorf("FileCount() = %d, want 2", got)
		}
	})

	t.Run("DriftCount counts only drifted files", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{
			{IsHeading: true},
			{Path: ".zshrc", SourceState: 'M', DestState: ' ', Drift: DriftSourceEdited},
			{Path: ".vimrc", SourceState: ' ', DestState: ' ', Drift: DriftNone},
		}
		if got := m.DriftCount(); got != 1 {
			t.Errorf("DriftCount() = %d, want 1", got)
		}
	})

	t.Run("ContentLines includes headings", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{
			{IsHeading: true},
			{Path: ".zshrc"},
			{Path: ".vimrc"},
		}
		if got := m.ContentLines(); got != 3 {
			t.Errorf("ContentLines() = %d, want 3", got)
		}
	})
}

func TestFileListModel_SetFiles(t *testing.T) {
	t.Run("clamps cursor when list shrinks", func(t *testing.T) {
		m := NewFileListModel()
		m.SetDimensions(80, 20)
		m.files = make([]FileItem, 10)
		m.cursor = 8

		// Set a smaller list (no drift so no headings inserted)
		smallList := []FileItem{
			{Path: ".a", SourceState: ' ', DestState: ' '},
			{Path: ".b", SourceState: ' ', DestState: ' '},
		}
		m.SetFiles(smallList)
		if m.cursor >= len(m.files) {
			t.Errorf("cursor = %d, should be < len(files)=%d", m.cursor, len(m.files))
		}
	})

	t.Run("snaps cursor off heading", func(t *testing.T) {
		m := NewFileListModel()
		m.SetDimensions(80, 20)

		// Files that will produce headings — cursor might land on one after clamp
		files := []FileItem{
			{Path: ".zshrc", SourceState: ' ', DestState: 'M', Drift: DriftDestEdited},
			{Path: ".vimrc", SourceState: ' ', DestState: ' ', Drift: DriftNone},
		}
		m.SetFiles(files)
		// After SetFiles, cursor should be on a file, not a heading
		if m.cursor < len(m.files) && m.files[m.cursor].IsHeading {
			t.Errorf("cursor at %d is on a heading", m.cursor)
		}
	})
}
