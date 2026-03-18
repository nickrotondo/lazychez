package ui

import "testing"

func TestIsNarrow(t *testing.T) {
	tests := []struct {
		width int
		want  bool
	}{
		{0, true},
		{84, true},
		{85, false},
		{120, false},
	}
	for _, tt := range tests {
		m := Model{width: tt.width}
		if got := m.isNarrow(); got != tt.want {
			t.Errorf("isNarrow() with width=%d = %v, want %v", tt.width, got, tt.want)
		}
	}
}

func TestDistributeLeftColumn(t *testing.T) {
	makeModel := func(focused PaneID, fileContentLines, gitContentLines int) Model {
		m, _, _ := newTestModel()
		m.focused = focused
		files := make([]FileItem, fileContentLines)
		for i := range files {
			files[i] = FileItem{Path: "f"}
		}
		m.fileList.files = files
		entries := make([]GitStatusEntry, gitContentLines)
		for i := range entries {
			entries[i] = GitStatusEntry{XY: " M", Path: "f"}
		}
		m.gitStatus.entries = entries
		return m
	}

	t.Run("even split regardless of focus", func(t *testing.T) {
		for _, focused := range []PaneID{PaneFileList, PaneGitStatus, PaneDiff} {
			m := makeModel(focused, 5, 3)
			fileH, gitH := m.distributeLeftColumn(30)
			if fileH != 15 {
				t.Errorf("focused=%d: fileH = %d, want 15", focused, fileH)
			}
			if gitH != 15 {
				t.Errorf("focused=%d: gitH = %d, want 15", focused, gitH)
			}
		}
	})

	t.Run("odd available rounds correctly", func(t *testing.T) {
		m := makeModel(PaneFileList, 5, 3)
		fileH, gitH := m.distributeLeftColumn(31)
		if fileH+gitH != 31 {
			t.Errorf("sum = %d, want 31", fileH+gitH)
		}
		if fileH != 15 || gitH != 16 {
			t.Errorf("fileH=%d gitH=%d, want 15 and 16", fileH, gitH)
		}
	})

	t.Run("ignores content lines", func(t *testing.T) {
		m := makeModel(PaneFileList, 100, 0)
		fileH, gitH := m.distributeLeftColumn(20)
		if fileH != 10 || gitH != 10 {
			t.Errorf("fileH=%d gitH=%d, want 10 and 10", fileH, gitH)
		}
	})

	t.Run("sum always equals available", func(t *testing.T) {
		configs := []struct {
			focused PaneID
			fileN   int
			gitN    int
			avail   int
		}{
			{PaneFileList, 5, 3, 30},
			{PaneGitStatus, 10, 10, 15},
			{PaneDiff, 0, 0, 20},
			{PaneFileList, 50, 50, 40},
		}
		for _, c := range configs {
			m := makeModel(c.focused, c.fileN, c.gitN)
			fileH, gitH := m.distributeLeftColumn(c.avail)
			if fileH+gitH != c.avail {
				t.Errorf("focused=%d file=%d git=%d avail=%d: sum=%d, want %d",
					c.focused, c.fileN, c.gitN, c.avail, fileH+gitH, c.avail)
			}
		}
	})
}

func TestDistributeNarrow(t *testing.T) {
	makeModel := func(focused PaneID, fileContentLines, gitContentLines int) Model {
		m, _, _ := newTestModel()
		m.focused = focused
		files := make([]FileItem, fileContentLines)
		for i := range files {
			files[i] = FileItem{Path: "f"}
		}
		m.fileList.files = files
		entries := make([]GitStatusEntry, gitContentLines)
		for i := range entries {
			entries[i] = GitStatusEntry{XY: " M", Path: "f"}
		}
		m.gitStatus.entries = entries
		return m
	}

	t.Run("file focused: git collapses, diff stays visible", func(t *testing.T) {
		m := makeModel(PaneFileList, 3, 3)
		fileH, gitH, diffH := m.distributeNarrow(60)
		if gitH != collapsedHeight {
			t.Errorf("gitH = %d, want %d", gitH, collapsedHeight)
		}
		// Diff should get roughly half, not collapse
		remaining := 60 - collapsedHeight
		wantFile := remaining / 2
		wantDiff := remaining - wantFile
		if fileH != wantFile {
			t.Errorf("fileH = %d, want %d", fileH, wantFile)
		}
		if diffH != wantDiff {
			t.Errorf("diffH = %d, want %d", diffH, wantDiff)
		}
	})

	t.Run("diff focused: prevFocused side pane stays expanded", func(t *testing.T) {
		m := makeModel(PaneDiff, 3, 3)
		m.prevFocused = PaneFileList
		fileH, gitH, diffH := m.distributeNarrow(60)
		// File list was prevFocused, so it stays expanded
		if gitH != collapsedHeight {
			t.Errorf("gitH = %d, want %d", gitH, collapsedHeight)
		}
		remaining := 60 - collapsedHeight
		wantFile := remaining / 2
		wantDiff := remaining - wantFile
		if fileH != wantFile {
			t.Errorf("fileH = %d, want %d", fileH, wantFile)
		}
		if diffH != wantDiff {
			t.Errorf("diffH = %d, want %d", diffH, wantDiff)
		}
	})

	t.Run("diff focused with prevFocused=git", func(t *testing.T) {
		m := makeModel(PaneDiff, 3, 3)
		m.prevFocused = PaneGitStatus
		fileH, gitH, diffH := m.distributeNarrow(60)
		if fileH != collapsedHeight {
			t.Errorf("fileH = %d, want %d", fileH, collapsedHeight)
		}
		remaining := 60 - collapsedHeight
		wantGit := remaining / 2
		wantDiff := remaining - wantGit
		if gitH != wantGit {
			t.Errorf("gitH = %d, want %d", gitH, wantGit)
		}
		if diffH != wantDiff {
			t.Errorf("diffH = %d, want %d", diffH, wantDiff)
		}
	})

	t.Run("git focused: file collapses, diff stays visible", func(t *testing.T) {
		m := makeModel(PaneGitStatus, 3, 3)
		fileH, gitH, diffH := m.distributeNarrow(40)
		if fileH != collapsedHeight {
			t.Errorf("fileH = %d, want %d", fileH, collapsedHeight)
		}
		remaining := 40 - collapsedHeight
		wantGit := remaining / 2
		wantDiff := remaining - wantGit
		if gitH != wantGit {
			t.Errorf("gitH = %d, want %d", gitH, wantGit)
		}
		if diffH != wantDiff {
			t.Errorf("diffH = %d, want %d", diffH, wantDiff)
		}
	})

	t.Run("sum always equals available", func(t *testing.T) {
		configs := []struct {
			focused PaneID
			fileN   int
			gitN    int
			avail   int
		}{
			{PaneFileList, 5, 3, 60},
			{PaneGitStatus, 10, 10, 30},
			{PaneDiff, 0, 0, 40},
			{PaneFileList, 50, 50, 50},
		}
		for _, c := range configs {
			m := makeModel(c.focused, c.fileN, c.gitN)
			fileH, gitH, diffH := m.distributeNarrow(c.avail)
			if fileH+gitH+diffH != c.avail {
				t.Errorf("focused=%d file=%d git=%d avail=%d: sum=%d, want %d",
					c.focused, c.fileN, c.gitN, c.avail, fileH+gitH+diffH, c.avail)
			}
		}
	})
}

func TestRenderCollapsedPane(t *testing.T) {
	t.Run("correct width without info", func(t *testing.T) {
		line := renderCollapsedPane("[2] Source Git", 40, "")
		if w := len([]rune(stripAnsi(line))); w != 40 {
			t.Errorf("collapsed pane width = %d, want 40", w)
		}
	})

	t.Run("correct width with info", func(t *testing.T) {
		line := renderCollapsedPane("[2] Source Git", 40, "1 of 5")
		if w := len([]rune(stripAnsi(line))); w != 40 {
			t.Errorf("collapsed pane width = %d, want 40", w)
		}
	})

	t.Run("contains title and info", func(t *testing.T) {
		line := renderCollapsedPane("[2] Source Git", 50, "1 of 5")
		stripped := stripAnsi(line)
		if !contains(stripped, "[2]") || !contains(stripped, "Source Git") {
			t.Errorf("collapsed pane missing title parts: %q", stripped)
		}
		if !contains(stripped, "1 of 5") {
			t.Errorf("collapsed pane missing info: %q", stripped)
		}
	})
}

// stripAnsi removes ANSI escape sequences for width testing.
func stripAnsi(s string) string {
	var out []byte
	i := 0
	for i < len(s) {
		if s[i] == '\033' {
			// Skip until 'm' (SGR) or end of sequence
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++ // skip 'm'
			}
		} else {
			out = append(out, s[i])
			i++
		}
	}
	return string(out)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
