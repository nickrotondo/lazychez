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

func TestClassifyDrift(t *testing.T) {
	tests := []struct {
		name  string
		entry chezmoi.StatusEntry
		want  DriftKind
	}{
		{"source edited: ' M'", chezmoi.StatusEntry{SourceState: ' ', DestState: 'M'}, DriftSourceEdited},
		{"source added: ' A'", chezmoi.StatusEntry{SourceState: ' ', DestState: 'A'}, DriftSourceEdited},
		{"dest edited: 'MM'", chezmoi.StatusEntry{SourceState: 'M', DestState: 'M'}, DriftDestEdited},
		{"dest edited: 'M '", chezmoi.StatusEntry{SourceState: 'M', DestState: ' '}, DriftDestEdited},
		{"dest added: 'A '", chezmoi.StatusEntry{SourceState: 'A', DestState: ' '}, DriftDestEdited},
		{"dest deleted: 'D '", chezmoi.StatusEntry{SourceState: 'D', DestState: ' '}, DriftDestEdited},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyDrift(tt.entry); got != tt.want {
				t.Errorf("classifyDrift() = %d, want %d", got, tt.want)
			}
		})
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
		items := MergeFilesWithStatus(managed, nil)
		if len(items) != 2 {
			t.Fatalf("got %d items, want 2", len(items))
		}
		for _, item := range items {
			if item.Drift != DriftNone {
				t.Errorf("item %q has drift %d, want DriftNone", item.Path, item.Drift)
			}
		}
	})

	t.Run("source edited via chezmoi edit", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		// chezmoi status ' M': dest unchanged, apply will modify → source was edited
		status := []chezmoi.StatusEntry{
			{SourceState: ' ', DestState: 'M', Path: ".zshrc"},
		}
		items := MergeFilesWithStatus(managed, status)
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
		if items[0].Drift != DriftSourceEdited {
			t.Errorf("drift = %d, want DriftSourceEdited", items[0].Drift)
		}
		if items[0].DestState != 'M' {
			t.Errorf("DestState = %c, want M", items[0].DestState)
		}
	})

	t.Run("dest edited directly", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		// chezmoi status 'MM': dest modified since last apply, apply will also modify → dest was edited
		status := []chezmoi.StatusEntry{
			{SourceState: 'M', DestState: 'M', Path: ".zshrc"},
		}
		items := MergeFilesWithStatus(managed, status)
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
		if items[0].Drift != DriftDestEdited {
			t.Errorf("drift = %d, want DriftDestEdited", items[0].Drift)
		}
	})

	t.Run("orphan status entry uses chezmoi columns", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		// Orphan with first col 'M' → dest edited
		status := []chezmoi.StatusEntry{
			{SourceState: 'M', DestState: 'A', Path: ".bashrc"},
		}
		items := MergeFilesWithStatus(managed, status)
		if len(items) != 2 {
			t.Fatalf("got %d items, want 2", len(items))
		}
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

	t.Run("orphan with source edit", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		// Orphan with first col ' ' → source edited
		status := []chezmoi.StatusEntry{
			{SourceState: ' ', DestState: 'A', Path: ".bashrc"},
		}
		items := MergeFilesWithStatus(managed, status)
		var orphan *FileItem
		for i := range items {
			if items[i].Path == ".bashrc" {
				orphan = &items[i]
			}
		}
		if orphan == nil {
			t.Fatal("orphan .bashrc not found")
		}
		if orphan.Drift != DriftSourceEdited {
			t.Errorf("orphan drift = %d, want DriftSourceEdited", orphan.Drift)
		}
	})

	t.Run("sort order dest then source then synced", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
			{Path: ".bashrc", SourceRelPath: "dot_bashrc"},
			{Path: ".vimrc", SourceRelPath: "dot_vimrc"},
		}
		status := []chezmoi.StatusEntry{
			{SourceState: ' ', DestState: 'M', Path: ".zshrc"},  // source edited (' M')
			{SourceState: 'M', DestState: 'M', Path: ".bashrc"}, // dest edited ('MM')
		}
		items := MergeFilesWithStatus(managed, status)

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
		items := MergeFilesWithStatus(managed, nil)
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
		items := MergeFilesWithStatus(nil, nil)
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
			{Path: ".zshrc", SourceState: 'M', DestState: 'M', Drift: DriftDestEdited},
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
			{Path: ".bashrc", Drift: DriftDestEdited, SourceState: 'M', DestState: 'M'},
			{Path: ".zshrc", Drift: DriftSourceEdited, SourceState: ' ', DestState: 'M'},
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

// makeFilterModel creates a FileListModel pre-loaded with files for filter tests.
// Files: .bashrc (dest edited), .zshrc (source edited), .vimrc (synced), .gitconfig (synced)
func makeFilterModel() FileListModel {
	m := NewFileListModel()
	m.SetDimensions(80, 20)
	m.SetFiles([]FileItem{
		{Path: ".bashrc", SourceRelPath: "dot_bashrc", SourceState: ' ', DestState: 'M', Drift: DriftDestEdited},
		{Path: ".zshrc", SourceRelPath: "dot_zshrc", SourceState: 'M', DestState: ' ', Drift: DriftSourceEdited},
		{Path: ".vimrc", SourceRelPath: "dot_vimrc", SourceState: ' ', DestState: ' ', Drift: DriftNone},
		{Path: ".gitconfig", SourceRelPath: "dot_gitconfig", SourceState: ' ', DestState: ' ', Drift: DriftNone},
	})
	return m
}

func TestFilter_FuzzyMatching(t *testing.T) {
	t.Run("exact substring match", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("zsh")
		m.applyFilter()

		if m.FileCount() != 1 {
			t.Fatalf("FileCount() = %d, want 1", m.FileCount())
		}
		if m.SelectedPath() != ".zshrc" {
			t.Errorf("SelectedPath() = %q, want .zshrc", m.SelectedPath())
		}
	})

	t.Run("fuzzy match across characters", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("brc")
		m.applyFilter()

		// Should match .bashrc and .zshrc (both contain b/r/c or similar fuzzy patterns)
		if m.FileCount() < 1 {
			t.Fatalf("FileCount() = %d, want >= 1", m.FileCount())
		}
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("ZSH")
		m.applyFilter()

		if m.FileCount() != 1 {
			t.Fatalf("FileCount() = %d, want 1", m.FileCount())
		}
		if m.SelectedPath() != ".zshrc" {
			t.Errorf("SelectedPath() = %q, want .zshrc", m.SelectedPath())
		}
	})

	t.Run("multiple matches", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("rc")
		m.applyFilter()

		// .bashrc, .zshrc, .vimrc all contain "rc"
		if m.FileCount() < 3 {
			t.Fatalf("FileCount() = %d, want >= 3", m.FileCount())
		}
	})
}

func TestFilter_EmptyQuery(t *testing.T) {
	m := makeFilterModel()
	totalBefore := m.FileCount()

	m.StartFilter()
	m.filterInput.SetValue("")
	m.applyFilter()

	if m.FileCount() != totalBefore {
		t.Errorf("FileCount() = %d, want %d (all items)", m.FileCount(), totalBefore)
	}
}

func TestFilter_NoMatches(t *testing.T) {
	m := makeFilterModel()
	m.StartFilter()
	m.filterInput.SetValue("zzzznotafile")
	m.applyFilter()

	if m.FileCount() != 0 {
		t.Errorf("FileCount() = %d, want 0", m.FileCount())
	}
	if m.FilterHasMatches() {
		t.Error("FilterHasMatches() = true, want false")
	}
}

func TestFilter_NavigationLocked(t *testing.T) {
	t.Run("j/k navigate only filtered items", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		// Match two items: .bashrc and .zshrc (both contain "sh")
		m.filterInput.SetValue("sh")
		m.applyFilter()

		if m.FileCount() != 2 {
			t.Fatalf("FileCount() = %d, want 2", m.FileCount())
		}

		locked := m.LockFilter()
		if !locked {
			t.Fatal("LockFilter() returned false")
		}

		// Cursor should be on first match
		first := m.SelectedPath()
		m.MoveDown()
		second := m.SelectedPath()

		if first == second {
			t.Error("MoveDown did not change selected path")
		}

		// MoveDown again should be no-op (only 2 items)
		m.MoveDown()
		if m.SelectedPath() != second {
			t.Errorf("MoveDown past last item changed path to %q", m.SelectedPath())
		}

		// MoveUp back to first
		m.MoveUp()
		if m.SelectedPath() != first {
			t.Errorf("MoveUp should return to %q, got %q", first, m.SelectedPath())
		}
	})
}

func TestFilter_EscDuringTyping(t *testing.T) {
	m := makeFilterModel()
	totalBefore := m.FileCount()
	m.cursor = 3 // put cursor on a specific item
	cursorBefore := m.cursor

	m.StartFilter()
	m.filterInput.SetValue("zsh")
	m.applyFilter()

	if m.FileCount() == totalBefore {
		t.Fatal("filter should have reduced visible items")
	}

	m.CancelFilter()

	if m.filterMode != FilterInactive {
		t.Errorf("filterMode = %d, want FilterInactive", m.filterMode)
	}
	if m.FileCount() != totalBefore {
		t.Errorf("FileCount() = %d, want %d (restored)", m.FileCount(), totalBefore)
	}
	if m.cursor != cursorBefore {
		t.Errorf("cursor = %d, want %d (restored)", m.cursor, cursorBefore)
	}
}

func TestFilter_EnterLocks(t *testing.T) {
	t.Run("lock with matches", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("vim")
		m.applyFilter()

		locked := m.LockFilter()
		if !locked {
			t.Fatal("LockFilter() returned false")
		}
		if m.filterMode != FilterLocked {
			t.Errorf("filterMode = %d, want FilterLocked", m.filterMode)
		}
		if !m.IsFilterLocked() {
			t.Error("IsFilterLocked() = false")
		}
		// Cursor should be on first match
		if m.SelectedPath() != ".vimrc" {
			t.Errorf("SelectedPath() = %q, want .vimrc", m.SelectedPath())
		}
	})

	t.Run("lock with no matches is no-op", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("zzzznotafile")
		m.applyFilter()

		locked := m.LockFilter()
		if locked {
			t.Error("LockFilter() should return false when no matches")
		}
		if m.filterMode != FilterTyping {
			t.Errorf("filterMode = %d, want FilterTyping (unchanged)", m.filterMode)
		}
	})
}

func TestFilter_EscDuringLocked(t *testing.T) {
	m := makeFilterModel()
	totalBefore := m.FileCount()

	m.StartFilter()
	m.filterInput.SetValue("vim")
	m.applyFilter()
	m.LockFilter()

	if m.IsFilterLocked() != true {
		t.Fatal("expected locked filter mode")
	}

	m.CancelFilter()

	if m.filterMode != FilterInactive {
		t.Errorf("filterMode = %d, want FilterInactive", m.filterMode)
	}
	if m.FileCount() != totalBefore {
		t.Errorf("FileCount() = %d, want %d", m.FileCount(), totalBefore)
	}
}

func TestFilter_ClearsOnDataRefresh(t *testing.T) {
	m := makeFilterModel()
	m.StartFilter()
	m.filterInput.SetValue("vim")
	m.applyFilter()
	m.LockFilter()

	// Simulate data refresh by calling SetFiles (which is what ManagedFilesMsg triggers)
	m.SetFiles([]FileItem{
		{Path: ".bashrc", SourceState: ' ', DestState: ' ', Drift: DriftNone},
		{Path: ".newfile", SourceState: ' ', DestState: ' ', Drift: DriftNone},
	})

	if m.filterMode != FilterInactive {
		t.Errorf("filterMode = %d, want FilterInactive after SetFiles", m.filterMode)
	}
	if m.FileCount() != 2 {
		t.Errorf("FileCount() = %d, want 2", m.FileCount())
	}
}

func TestFilter_GroupHeadingsHidden(t *testing.T) {
	m := makeFilterModel()
	m.StartFilter()
	// Match only synced files (no drift)
	m.filterInput.SetValue("vim")
	m.applyFilter()

	// Only .vimrc should match — it's in the "synced" group
	if m.FileCount() != 1 {
		t.Fatalf("FileCount() = %d, want 1", m.FileCount())
	}

	// Check that no dest-edited or source-edited headings appear
	for _, f := range m.files {
		if f.IsHeading && f.HeadingText != headingText(DriftNone) {
			t.Errorf("unexpected heading %q visible in filtered list", f.HeadingText)
		}
	}
}

func TestFilter_FileOperationsOnFilteredItems(t *testing.T) {
	m := makeFilterModel()
	m.StartFilter()
	m.filterInput.SetValue("bashrc")
	m.applyFilter()
	m.LockFilter()

	// SelectedItem should return the filtered item, not a different one
	item := m.SelectedItem()
	if item == nil {
		t.Fatal("SelectedItem() = nil")
	}
	if item.Path != ".bashrc" {
		t.Errorf("SelectedItem().Path = %q, want .bashrc", item.Path)
	}
	if item.Drift != DriftDestEdited {
		t.Errorf("SelectedItem().Drift = %d, want DriftDestEdited", item.Drift)
	}
}

func TestFilter_ModeTransitions(t *testing.T) {
	t.Run("inactive → typing → locked → inactive", func(t *testing.T) {
		m := makeFilterModel()

		if m.filterMode != FilterInactive {
			t.Fatalf("initial filterMode = %d, want FilterInactive", m.filterMode)
		}

		m.StartFilter()
		if m.filterMode != FilterTyping {
			t.Fatalf("after StartFilter: filterMode = %d, want FilterTyping", m.filterMode)
		}
		if !m.IsFiltering() {
			t.Error("IsFiltering() = false after StartFilter")
		}

		m.filterInput.SetValue("vim")
		m.applyFilter()
		m.LockFilter()
		if m.filterMode != FilterLocked {
			t.Fatalf("after LockFilter: filterMode = %d, want FilterLocked", m.filterMode)
		}

		m.CancelFilter()
		if m.filterMode != FilterInactive {
			t.Fatalf("after CancelFilter: filterMode = %d, want FilterInactive", m.filterMode)
		}
	})

	t.Run("inactive → typing → cancel → inactive", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("zsh")
		m.applyFilter()

		m.CancelFilter()
		if m.filterMode != FilterInactive {
			t.Errorf("filterMode = %d, want FilterInactive", m.filterMode)
		}
		if m.IsFiltering() {
			t.Error("IsFiltering() = true after cancel")
		}
		if m.IsFilterLocked() {
			t.Error("IsFilterLocked() = true after cancel")
		}
	})
}

func TestFilter_CursorPosition(t *testing.T) {
	m := makeFilterModel()
	m.StartFilter()
	m.filterInput.SetValue("sh")
	m.applyFilter()
	m.LockFilter()

	current, total := m.CursorPosition()
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if current != 1 {
		t.Errorf("current = %d, want 1 (first match)", current)
	}

	m.MoveDown()
	current, total = m.CursorPosition()
	if current != 2 {
		t.Errorf("after MoveDown: current = %d, want 2", current)
	}
	if total != 2 {
		t.Errorf("after MoveDown: total = %d, want 2", total)
	}
}
