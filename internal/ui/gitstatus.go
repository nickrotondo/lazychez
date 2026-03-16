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

func (m GitStatusModel) EntryCount() int {
	return len(m.entries)
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

		text := " " + e.Path
		xyWidth := lipgloss.Width(xyStr)
		if pad := m.width - xyWidth - lipgloss.Width(text); pad > 0 {
			text += strings.Repeat(" ", pad)
		}

		pathStyle := filePathStyle(e.XY)
		if selected {
			pathStyle = pathStyle.Background(SelectedBg).Bold(true)
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
