package ui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickrotondo/lazychez/internal/chezmoi"
)

type DriftKind int

const (
	DriftNone         DriftKind = iota
	DriftSourceEdited           // source changed (chezmoi edit) → suggest apply
	DriftDestEdited             // destination changed directly → suggest add
)

type FileItem struct {
	Path          string
	SourceRelPath string
	SourceState   rune // 'M', 'A', 'D', ' '
	DestState     rune
	Drift         DriftKind
}

func (f FileItem) HasDrift() bool {
	return f.SourceState != ' ' || f.DestState != ' '
}

type FileListModel struct {
	files   []FileItem
	cursor  int
	offset  int // scroll offset for viewport
	focused bool
	width   int
	height  int
}

func NewFileListModel() FileListModel {
	return FileListModel{}
}

func (m *FileListModel) SetFiles(files []FileItem) {
	m.files = files
	// Clamp cursor if file list shrunk
	if m.cursor >= len(m.files) {
		m.cursor = max(0, len(m.files)-1)
	}
	m.clampOffset()
}

func (m *FileListModel) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	m.clampOffset()
}

func (m *FileListModel) SetFocused(focused bool) {
	m.focused = focused
}

func (m FileListModel) SelectedPath() string {
	if len(m.files) == 0 {
		return ""
	}
	return m.files[m.cursor].Path
}

func (m FileListModel) FileCount() int {
	return len(m.files)
}

// ContentLines returns the number of lines the file list content needs
// (not counting borders or title).
func (m FileListModel) ContentLines() int {
	return len(m.files)
}

func (m FileListModel) DriftCount() int {
	count := 0
	for _, f := range m.files {
		if f.HasDrift() {
			count++
		}
	}
	return count
}

// MoveUp moves cursor up, clamping at top.
func (m *FileListModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
		m.clampOffset()
	}
}

// MoveDown moves cursor down, clamping at bottom.
func (m *FileListModel) MoveDown() {
	if m.cursor < len(m.files)-1 {
		m.cursor++
		m.clampOffset()
	}
}

// GoToTop jumps cursor to first item.
func (m *FileListModel) GoToTop() {
	m.cursor = 0
	m.clampOffset()
}

// GoToBottom jumps cursor to last item.
func (m *FileListModel) GoToBottom() {
	if len(m.files) > 0 {
		m.cursor = len(m.files) - 1
		m.clampOffset()
	}
}

func (m *FileListModel) clampOffset() {
	visible := m.visibleLines()
	if visible <= 0 {
		return
	}
	// Keep cursor visible with 2 lines of scroll padding
	const padding = 2
	if m.cursor < m.offset+padding {
		m.offset = max(0, m.cursor-padding)
	}
	if m.cursor >= m.offset+visible-padding {
		m.offset = max(0, m.cursor-visible+padding+1)
	}
	// Don't scroll past content
	maxOffset := max(0, len(m.files)-visible)
	m.offset = min(m.offset, maxOffset)
}

func (m FileListModel) visibleLines() int {
	return m.height
}

func (m FileListModel) View() string {
	if len(m.files) == 0 {
		return lipgloss.NewStyle().
			Foreground(MutedColor).
			Render("No managed files")
	}

	visible := m.visibleLines()
	end := min(m.offset+visible, len(m.files))

	var lines []string
	for i := m.offset; i < end; i++ {
		f := m.files[i]
		selected := i == m.cursor && m.focused

		// Determine indicator style — when selected, merge with selection background
		// so the inner ANSI reset doesn't kill the highlight.
		var indicatorStr string
		switch {
		case f.Drift == DriftSourceEdited:
			if selected {
				indicatorStr = lipgloss.NewStyle().Foreground(ModifiedColor).Background(SelectedBg).Bold(true).Render("●") + " "
			} else {
				indicatorStr = SourceEditedIndicator.String() + " "
			}
		case f.Drift == DriftDestEdited:
			if selected {
				indicatorStr = lipgloss.NewStyle().Foreground(TitleColor).Background(SelectedBg).Bold(true).Render("◆") + " "
			} else {
				indicatorStr = DestEditedIndicator.String() + " "
			}
		case f.SourceState == 'A' || f.DestState == 'A':
			if selected {
				indicatorStr = lipgloss.NewStyle().Foreground(AddedColor).Background(SelectedBg).Bold(true).Render("+") + " "
			} else {
				indicatorStr = AddedIndicator.String() + " "
			}
		case f.SourceState == 'D' || f.DestState == 'D':
			if selected {
				indicatorStr = lipgloss.NewStyle().Foreground(DeletedColor).Background(SelectedBg).Bold(true).Render("−") + " "
			} else {
				indicatorStr = DeletedIndicator.String() + " "
			}
		default:
			indicatorStr = "  "
		}

		text := f.Path
		// Pad to full width
		visIndicator := lipgloss.Width(indicatorStr)
		if pad := m.width - visIndicator - len(text); pad > 0 {
			text += strings.Repeat(" ", pad)
		}

		if selected {
			line := indicatorStr + SelectedItem.Render(text)
			lines = append(lines, line)
		} else {
			lines = append(lines, indicatorStr+text)
		}
	}

	return strings.Join(lines, "\n")
}

// MergeFilesWithStatus builds the merged and sorted file list.
// gitModifiedPaths is a set of source-relative paths that are dirty in git.
func MergeFilesWithStatus(managed []chezmoi.ManagedFile, status []chezmoi.StatusEntry, gitModifiedPaths map[string]bool) []FileItem {
	statusMap := make(map[string]chezmoi.StatusEntry, len(status))
	for _, s := range status {
		statusMap[s.Path] = s
	}

	items := make([]FileItem, 0, len(managed))
	for _, f := range managed {
		item := FileItem{Path: f.Path, SourceRelPath: f.SourceRelPath, SourceState: ' ', DestState: ' '}
		if s, ok := statusMap[f.Path]; ok {
			item.SourceState = s.SourceState
			item.DestState = s.DestState
			// Classify drift direction: if the source file is dirty in git,
			// it was edited via chezmoi (suggest apply). Otherwise the
			// destination was edited directly (suggest add).
			if gitModifiedPaths[f.SourceRelPath] {
				item.Drift = DriftSourceEdited
			} else {
				item.Drift = DriftDestEdited
			}
		}
		items = append(items, item)
	}

	// Add status entries not in managed (e.g. deleted files)
	managedSet := make(map[string]bool, len(managed))
	for _, f := range managed {
		managedSet[f.Path] = true
	}
	for _, s := range status {
		if !managedSet[s.Path] {
			items = append(items, FileItem{
				Path:        s.Path,
				SourceState: s.SourceState,
				DestState:   s.DestState,
				Drift:       DriftDestEdited,
			})
		}
	}

	// Sort: drifted files first, then alphabetical
	sort.SliceStable(items, func(i, j int) bool {
		di, dj := items[i].HasDrift(), items[j].HasDrift()
		if di != dj {
			return di
		}
		return items[i].Path < items[j].Path
	})

	return items
}
