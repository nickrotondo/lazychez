package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type DiffViewModel struct {
	viewport viewport.Model
	path     string
	rawDiff  string
	focused  bool
	ready    bool
}

func NewDiffViewModel() DiffViewModel {
	return DiffViewModel{}
}

func (m *DiffViewModel) SetDimensions(w, h int) {
	if !m.ready {
		m.viewport = viewport.New(w, h)
		m.ready = true
	} else {
		m.viewport.Width = w
		m.viewport.Height = h
	}
	// Re-render with current content at new dimensions
	if m.rawDiff != "" {
		m.viewport.SetContent(colorizeDiff(m.rawDiff))
	}
}

func (m *DiffViewModel) SetContent(path, diff string) {
	m.path = path
	m.rawDiff = diff
	if m.ready {
		m.viewport.SetContent(colorizeDiff(diff))
		m.viewport.GotoTop()
	}
}

func (m *DiffViewModel) SetFocused(focused bool) {
	m.focused = focused
}

// ScrollState returns the scroll offset and total line count for scrollbar rendering.
func (m DiffViewModel) ScrollState() (offset, total int) {
	if !m.ready || m.rawDiff == "" {
		return 0, 0
	}
	return m.viewport.YOffset, m.viewport.TotalLineCount()
}

func (m DiffViewModel) Path() string {
	return m.path
}

func (m DiffViewModel) Update(msg tea.Msg) (DiffViewModel, tea.Cmd) {
	if !m.ready || !m.focused {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DiffViewModel) View() string {
	if m.rawDiff == "" {
		return NormalItem.Foreground(MutedColor).Render("Select a file to view diff")
	}
	if !m.ready {
		return ""
	}
	return m.viewport.View()
}

func colorizeDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var colored []string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			colored = append(colored, DiffMeta.Render(line))
		case strings.HasPrefix(line, "@@"):
			colored = append(colored, DiffHunk.Render(line))
		case strings.HasPrefix(line, "+"):
			colored = append(colored, DiffAdd.Render(line))
		case strings.HasPrefix(line, "-"):
			colored = append(colored, DiffDel.Render(line))
		case strings.HasPrefix(line, "diff "):
			colored = append(colored, DiffMeta.Render(line))
		default:
			colored = append(colored, line)
		}
	}
	return strings.Join(colored, "\n")
}
