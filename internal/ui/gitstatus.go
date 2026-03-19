package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type GitStatusEntry struct {
	XY   string // e.g. "M ", "??"
	Path string
}

type GitStatusModel struct {
	entries []GitStatusEntry
	cursor  int
	offset  int
	focused bool
	width   int
	height  int
}

func NewGitStatusModel() GitStatusModel {
	return GitStatusModel{}
}

func (m *GitStatusModel) SetEntries(entries []GitStatusEntry) {
	m.entries = entries
	if m.cursor >= len(m.entries) {
		m.cursor = max(0, len(m.entries)-1)
	}
	m.clampOffset()
}

func (m *GitStatusModel) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	m.clampOffset()
}

func (m *GitStatusModel) SetFocused(focused bool) {
	m.focused = focused
}

func (m GitStatusModel) SelectedPath() string {
	if len(m.entries) == 0 {
		return ""
	}
	return m.entries[m.cursor].Path
}

func (m GitStatusModel) SelectedEntry() (GitStatusEntry, bool) {
	if len(m.entries) == 0 {
		return GitStatusEntry{}, false
	}
	return m.entries[m.cursor], true
}

// ScrollState returns the scroll offset and total line count for scrollbar rendering.
func (m GitStatusModel) ScrollState() (offset, total int) {
	return m.offset, len(m.entries)
}

// CursorPosition returns the 1-based cursor index and total entry count.
func (m GitStatusModel) CursorPosition() (current, total int) {
	total = len(m.entries)
	if total > 0 {
		current = m.cursor + 1
	}
	return current, total
}

func (m GitStatusModel) EntryCount() int {
	return len(m.entries)
}

func (m GitStatusModel) HasStagedFiles() bool {
	for _, e := range m.entries {
		if len(e.XY) >= 2 && e.XY[0] != ' ' && e.XY[0] != '?' {
			return true
		}
	}
	return false
}

func (m GitStatusModel) ContentLines() int {
	return len(m.entries)
}

func (m *GitStatusModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
		m.clampOffset()
	}
}

func (m *GitStatusModel) MoveDown() {
	if m.cursor < len(m.entries)-1 {
		m.cursor++
		m.clampOffset()
	}
}

func (m *GitStatusModel) GoToTop() {
	m.cursor = 0
	m.clampOffset()
}

func (m *GitStatusModel) GoToBottom() {
	if len(m.entries) > 0 {
		m.cursor = len(m.entries) - 1
		m.clampOffset()
	}
}

// HalfPageDown moves cursor down by half the visible height.
func (m *GitStatusModel) HalfPageDown() {
	n := max(1, m.height/2)
	for i := 0; i < n; i++ {
		m.MoveDown()
	}
}

// HalfPageUp moves cursor up by half the visible height.
func (m *GitStatusModel) HalfPageUp() {
	n := max(1, m.height/2)
	for i := 0; i < n; i++ {
		m.MoveUp()
	}
}

func (m *GitStatusModel) clampOffset() {
	visible := m.height
	if visible <= 0 {
		return
	}
	const padding = 2
	if m.cursor < m.offset+padding {
		m.offset = max(0, m.cursor-padding)
	}
	if m.cursor >= m.offset+visible-padding {
		m.offset = max(0, m.cursor-visible+padding+1)
	}
	maxOffset := max(0, len(m.entries)-visible)
	m.offset = min(m.offset, maxOffset)
}

func (m GitStatusModel) View() string {
	if len(m.entries) == 0 {
		return lipgloss.NewStyle().
			Foreground(MutedColor).
			Render("Clean working tree")
	}

	visible := m.height
	end := min(m.offset+visible, len(m.entries))

	var lines []string
	for i := m.offset; i < end; i++ {
		e := m.entries[i]
		selected := i == m.cursor && m.focused

		xyStr := colorizeXY(e.XY, selected)

		xyWidth := lipgloss.Width(xyStr)
		pathStyle := filePathStyle(e.XY)
		if selected {
			pathStyle = pathStyle.Background(SelectedBg).Bold(true)
		}

		pathMax := m.width - xyWidth - 1
		if pathMax <= 0 {
			lines = append(lines, xyStr+pathStyle.Render(""))
			continue
		}

		var text string
		if len(e.Path) <= pathMax {
			text = " " + e.Path
			if pad := m.width - xyWidth - len(text); pad > 0 {
				text += strings.Repeat(" ", pad)
			}
		} else if m.focused {
			indentStr := strings.Repeat(" ", xyWidth+1)
			var b strings.Builder
			remaining := e.Path
			first := true
			for len(remaining) > 0 {
				n := min(len(remaining), pathMax)
				chunk := remaining[:n]
				remaining = remaining[n:]
				if first {
					b.WriteString(" " + chunk)
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
			text = " " + e.Path[:pathMax-1] + "…"
			if pad := m.width - xyWidth - lipgloss.Width(text); pad > 0 {
				text += strings.Repeat(" ", pad)
			}
		}

		lines = append(lines, xyStr+pathStyle.Render(text))
	}

	return strings.Join(lines, "\n")
}

func filePathStyle(xy string) lipgloss.Style {
	if isFullyStaged(xy) {
		return lipgloss.NewStyle().Foreground(AddedColor)
	}
	return lipgloss.NewStyle()
}

func isFullyStaged(xy string) bool {
	return len(xy) >= 2 && xy[1] == ' ' && xy[0] != ' ' && xy[0] != '?'
}

func colorizeXY(xy string, selected bool) string {
	if len(xy) < 2 {
		return xy
	}
	base := lipgloss.NewStyle()
	if selected {
		base = base.Background(SelectedBg).Bold(true)
	}
	x := renderPositionChar(rune(xy[0]), true, base)
	y := renderPositionChar(rune(xy[1]), false, base)
	return x + y
}

// renderPositionChar colors a status char green if in the index (staged)
// or red if in the worktree (unstaged).
func renderPositionChar(c rune, staged bool, base lipgloss.Style) string {
	if c == ' ' || c == 0 {
		return base.Render(string(c))
	}
	if staged {
		return base.Foreground(AddedColor).Render(string(c))
	}
	return base.Foreground(DeletedColor).Render(string(c))
}
