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
		// Set files directly to control ContentLines()
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

	t.Run("surplus to file list when file list focused", func(t *testing.T) {
		m := makeModel(PaneFileList, 5, 3)
		fileH, gitH := m.distributeLeftColumn(30)
		gitExpected := 3 + paneChrome // git gets its content + chrome
		if gitH != gitExpected {
			t.Errorf("gitH = %d, want %d", gitH, gitExpected)
		}
		if fileH+gitH != 30 {
			t.Errorf("fileH(%d) + gitH(%d) = %d, want 30", fileH, gitH, fileH+gitH)
		}
		// file should get surplus
		if fileH <= 5+paneChrome {
			t.Errorf("fileH = %d, should be > %d (content+chrome) due to surplus", fileH, 5+paneChrome)
		}
	})

	t.Run("surplus to git when git focused", func(t *testing.T) {
		m := makeModel(PaneGitStatus, 5, 3)
		fileH, gitH := m.distributeLeftColumn(30)
		fileExpected := 5 + paneChrome
		if fileH != fileExpected {
			t.Errorf("fileH = %d, want %d", fileH, fileExpected)
		}
		if fileH+gitH != 30 {
			t.Errorf("sum = %d, want 30", fileH+gitH)
		}
	})

	t.Run("surplus to file list when diff focused", func(t *testing.T) {
		m := makeModel(PaneDiff, 5, 3)
		fileH, gitH := m.distributeLeftColumn(30)
		gitExpected := 3 + paneChrome
		if gitH != gitExpected {
			t.Errorf("gitH = %d, want %d", gitH, gitExpected)
		}
		if fileH+gitH != 30 {
			t.Errorf("sum = %d, want 30", fileH+gitH)
		}
	})

	t.Run("cap at 70 percent", func(t *testing.T) {
		m := makeModel(PaneFileList, 100, 100)
		fileH, gitH := m.distributeLeftColumn(20)
		maxH := 20 * 70 / 100
		// In over-budget scenario, each pane should be reasonable
		if fileH+gitH != 20 {
			t.Errorf("sum = %d, want 20", fileH+gitH)
		}
		// Neither should exceed 70% of available
		if fileH > maxH+1 || gitH > maxH+1 {
			t.Errorf("fileH=%d or gitH=%d exceeds max %d", fileH, gitH, maxH)
		}
	})

	t.Run("minimum chrome", func(t *testing.T) {
		m := makeModel(PaneFileList, 0, 0)
		fileH, gitH := m.distributeLeftColumn(10)
		if fileH < paneChrome {
			t.Errorf("fileH = %d, should be >= paneChrome(%d)", fileH, paneChrome)
		}
		if gitH < paneChrome {
			t.Errorf("gitH = %d, should be >= paneChrome(%d)", gitH, paneChrome)
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

	t.Run("surplus to file list when focused", func(t *testing.T) {
		m := makeModel(PaneFileList, 3, 3)
		fileH, gitH, diffH := m.distributeNarrow(60)
		if fileH <= gitH {
			t.Errorf("fileH(%d) should be > gitH(%d) since file list is focused", fileH, gitH)
		}
		if fileH+gitH+diffH != 60 {
			t.Errorf("sum = %d, want 60", fileH+gitH+diffH)
		}
	})

	t.Run("surplus to diff when focused", func(t *testing.T) {
		m := makeModel(PaneDiff, 3, 3)
		fileH, gitH, diffH := m.distributeNarrow(60)
		if diffH <= fileH || diffH <= gitH {
			t.Errorf("diffH(%d) should be > fileH(%d) and gitH(%d)", diffH, fileH, gitH)
		}
		if fileH+gitH+diffH != 60 {
			t.Errorf("sum = %d, want 60", fileH+gitH+diffH)
		}
	})

	t.Run("minimum chrome for all panes", func(t *testing.T) {
		m := makeModel(PaneFileList, 0, 0)
		fileH, gitH, diffH := m.distributeNarrow(10)
		if fileH < paneChrome {
			t.Errorf("fileH = %d, should be >= %d", fileH, paneChrome)
		}
		if gitH < paneChrome {
			t.Errorf("gitH = %d, should be >= %d", gitH, paneChrome)
		}
		if diffH < paneChrome {
			t.Errorf("diffH = %d, should be >= %d", diffH, paneChrome)
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
