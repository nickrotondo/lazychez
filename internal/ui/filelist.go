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

// sortOrder returns the group priority for sorting: dest-edited first,
// then source-edited, then synced.
func (d DriftKind) sortOrder() int {
	switch d {
	case DriftDestEdited:
		return 0
	case DriftSourceEdited:
		return 1
	default:
		return 2
	}
}

type FileItem struct {
	Path          string
	SourceRelPath string
	SourceState   rune // 'M', 'A', 'D', ' '
	DestState     rune
	Drift         DriftKind
	IsHeading     bool
	HeadingText   string
}

func (f FileItem) HasDrift() bool {
	return !f.IsHeading && (f.SourceState != ' ' || f.DestState != ' ')
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
	m.files = insertHeadings(files)
	// Clamp cursor if file list shrunk
	if m.cursor >= len(m.files) {
		m.cursor = max(0, len(m.files)-1)
	}
	m.snapCursorToFile()
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
	if len(m.files) == 0 || m.files[m.cursor].IsHeading {
		return ""
	}
	return m.files[m.cursor].Path
}

func (m FileListModel) SelectedItem() *FileItem {
	if len(m.files) == 0 || m.cursor >= len(m.files) || m.files[m.cursor].IsHeading {
		return nil
	}
	return &m.files[m.cursor]
}

func (m FileListModel) FileCount() int {
	count := 0
	for _, f := range m.files {
		if !f.IsHeading {
			count++
		}
	}
	return count
}

// ContentLines returns the number of lines the file list content needs
// (not counting borders or title). Includes heading lines.
func (m FileListModel) ContentLines() int {
	return len(m.files)
}

// ScrollState returns the scroll offset and total line count for scrollbar rendering.
func (m FileListModel) ScrollState() (offset, total int) {
	return m.offset, len(m.files)
}

// CursorPosition returns the 1-based index among non-heading items and the total count.
func (m FileListModel) CursorPosition() (current, total int) {
	pos := 0
	for i, f := range m.files {
		if f.IsHeading {
			continue
		}
		pos++
		if i == m.cursor {
			current = pos
		}
	}
	return current, pos
}

func (m FileListModel) DriftCount() int {
	count := 0
	for _, f := range m.files {
		if !f.IsHeading && f.HasDrift() {
			count++
		}
	}
	return count
}

// MoveUp moves cursor up, skipping headings.
func (m *FileListModel) MoveUp() {
	for i := m.cursor - 1; i >= 0; i-- {
		if !m.files[i].IsHeading {
			m.cursor = i
			m.clampOffset()
			return
		}
	}
}

// MoveDown moves cursor down, skipping headings.
func (m *FileListModel) MoveDown() {
	for i := m.cursor + 1; i < len(m.files); i++ {
		if !m.files[i].IsHeading {
			m.cursor = i
			m.clampOffset()
			return
		}
	}
}

// GoToTop jumps cursor to first non-heading item.
func (m *FileListModel) GoToTop() {
	m.cursor = 0
	m.skipForward()
	m.clampOffset()
}

// GoToBottom jumps cursor to last non-heading item.
func (m *FileListModel) GoToBottom() {
	if len(m.files) > 0 {
		m.cursor = len(m.files) - 1
		m.skipBackward()
		m.clampOffset()
	}
}

// HalfPageDown moves cursor down by half the visible height, skipping headings.
func (m *FileListModel) HalfPageDown() {
	n := max(1, m.visibleLines()/2)
	for i := 0; i < n; i++ {
		m.MoveDown()
	}
}

// HalfPageUp moves cursor up by half the visible height, skipping headings.
func (m *FileListModel) HalfPageUp() {
	n := max(1, m.visibleLines()/2)
	for i := 0; i < n; i++ {
		m.MoveUp()
	}
}

// snapCursorToFile ensures cursor is on a file, not a heading.
func (m *FileListModel) snapCursorToFile() {
	if len(m.files) == 0 {
		return
	}
	if !m.files[m.cursor].IsHeading {
		return
	}
	// Try forward first, then backward
	saved := m.cursor
	m.skipForward()
	if m.cursor < len(m.files) && !m.files[m.cursor].IsHeading {
		return
	}
	m.cursor = saved
	m.skipBackward()
}

func (m *FileListModel) skipForward() {
	for m.cursor < len(m.files) && m.files[m.cursor].IsHeading {
		m.cursor++
	}
	if m.cursor >= len(m.files) && len(m.files) > 0 {
		m.cursor = len(m.files) - 1
	}
}

func (m *FileListModel) skipBackward() {
	for m.cursor > 0 && m.files[m.cursor].IsHeading {
		m.cursor--
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

		if f.IsHeading {
			lines = append(lines, renderHeadingLine(f.HeadingText, m.width))
			continue
		}

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

		visIndicator := lipgloss.Width(indicatorStr)
		pathMax := m.width - visIndicator

		var text string
		if pathMax <= 0 {
			text = ""
		} else if len(f.Path) <= pathMax {
			text = f.Path
			if pad := pathMax - len(text); pad > 0 {
				text += strings.Repeat(" ", pad)
			}
		} else if m.focused {
			indentStr := strings.Repeat(" ", visIndicator)
			var b strings.Builder
			remaining := f.Path
			first := true
			for len(remaining) > 0 {
				n := min(len(remaining), pathMax)
				chunk := remaining[:n]
				remaining = remaining[n:]
				if first {
					b.WriteString(chunk)
					first = false
				} else {
					b.WriteString("\n" + indentStr + chunk)
				}
				if pad := pathMax - len(chunk); pad > 0 {
					b.WriteString(strings.Repeat(" ", pad))
				}
			}
			text = b.String()
		} else {
			if pathMax > 1 {
				text = f.Path[:pathMax-1] + "…"
			}
			if pad := pathMax - len(text); pad > 0 {
				text += strings.Repeat(" ", pad)
			}
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

	// Sort: dest-edited first, then source-edited, then synced; alphabetical within each group
	sort.SliceStable(items, func(i, j int) bool {
		oi, oj := items[i].Drift.sortOrder(), items[j].Drift.sortOrder()
		if oi != oj {
			return oi < oj
		}
		return items[i].Path < items[j].Path
	})

	return items
}

// insertHeadings adds section heading entries at group boundaries.
// Only inserts headings when at least one file has drift.
func insertHeadings(files []FileItem) []FileItem {
	hasDrift := false
	for _, f := range files {
		if f.HasDrift() {
			hasDrift = true
			break
		}
	}
	if !hasDrift {
		return files
	}

	result := make([]FileItem, 0, len(files)+3)
	lastOrder := -1
	for _, f := range files {
		order := f.Drift.sortOrder()
		if order != lastOrder {
			result = append(result, FileItem{
				IsHeading:   true,
				HeadingText: headingText(f.Drift),
			})
			lastOrder = order
		}
		result = append(result, f)
	}
	return result
}

func headingText(d DriftKind) string {
	switch d {
	case DriftDestEdited:
		return "dest edited · space to add"
	case DriftSourceEdited:
		return "source edited · a to apply"
	default:
		return "synced"
	}
}

func renderHeadingLine(text string, width int) string {
	prefix := "── "
	suffix := " "
	inner := prefix + text + suffix
	fill := max(0, width-lipgloss.Width(inner))
	return lipgloss.NewStyle().Foreground(MutedColor).Render(inner + strings.Repeat("─", fill))
}
