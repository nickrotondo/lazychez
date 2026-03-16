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

		var xyStr string
		if selected {
			xyStr = colorizeXYSelected(e.XY)
		} else {
			xyStr = colorizeXY(e.XY)
		}

		text := " " + e.Path
		xyWidth := lipgloss.Width(xyStr)
		if pad := m.width - xyWidth - lipgloss.Width(text); pad > 0 {
			text += strings.Repeat(" ", pad)
		}

		if selected {
			lines = append(lines, xyStr+SelectedItem.Render(text))
		} else {
			lines = append(lines, xyStr+text)
		}
	}

	return strings.Join(lines, "\n")
}

func colorizeXYSelected(xy string) string {
	if len(xy) < 2 {
		return xy
	}
	x := colorizeStatusCharSelected(rune(xy[0]))
	y := colorizeStatusCharSelected(rune(xy[1]))
	return x + y
}

func colorizeXY(xy string) string {
	if len(xy) < 2 {
		return xy
	}
	x := colorizeStatusChar(rune(xy[0]))
	y := colorizeStatusChar(rune(xy[1]))
	return x + y
}

func colorizeStatusChar(c rune) string {
	switch c {
	case 'M':
		return lipgloss.NewStyle().Foreground(ModifiedColor).Render(string(c))
	case 'A':
		return lipgloss.NewStyle().Foreground(AddedColor).Render(string(c))
	case 'D':
		return lipgloss.NewStyle().Foreground(DeletedColor).Render(string(c))
	case '?':
		return lipgloss.NewStyle().Foreground(AddedColor).Render(string(c))
	default:
		return string(c)
	}
}

func colorizeStatusCharSelected(c rune) string {
	base := lipgloss.NewStyle().Background(SelectedBg).Bold(true)
	switch c {
	case 'M':
		return base.Foreground(ModifiedColor).Render(string(c))
	case 'A':
		return base.Foreground(AddedColor).Render(string(c))
	case 'D':
		return base.Foreground(DeletedColor).Render(string(c))
	case '?':
		return base.Foreground(AddedColor).Render(string(c))
	default:
		return base.Render(string(c))
	}
}
