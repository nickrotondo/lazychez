package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

const twoColumnBreakpoint = 70

// AddFileModel presents unmanaged files with a text filter and navigable list.
// Renders in 1 or 2 columns based on available width. The filter input is
// auto-focused; arrow keys navigate, typing filters.
type AddFileModel struct {
	filter   textinput.Model
	allFiles []string
	filtered []string
	cursor   int
	offset   int
	width    int
	height   int
	columns  int
}

func NewAddFileModel(files []string, width, height int) AddFileModel {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.Prompt = "/ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ActiveBorderColor)
	ti.Focus()

	columns := 1
	if width >= twoColumnBreakpoint {
		columns = 2
	}

	return AddFileModel{
		filter:   ti,
		allFiles: files,
		filtered: files,
		width:    width,
		height:   height,
		columns:  columns,
	}
}

func (m AddFileModel) SelectedPath() string {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return ""
	}
	return m.filtered[m.cursor]
}

func (m AddFileModel) Update(msg tea.Msg) (AddFileModel, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.Type {
		case tea.KeyUp:
			m.moveUp()
			return m, nil
		case tea.KeyDown:
			m.moveDown()
			return m, nil
		case tea.KeyPgUp:
			m.pageUp()
			return m, nil
		case tea.KeyPgDown:
			m.pageDown()
			return m, nil
		case tea.KeyHome:
			m.goToTop()
			return m, nil
		case tea.KeyEnd:
			m.goToBottom()
			return m, nil
		}
	}

	prevValue := m.filter.Value()
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	if m.filter.Value() != prevValue {
		m.applyFilter()
	}
	return m, cmd
}

func (m *AddFileModel) applyFilter() {
	query := m.filter.Value()
	if query == "" {
		m.filtered = m.allFiles
	} else {
		matches := fuzzy.Find(query, m.allFiles)
		filtered := make([]string, len(matches))
		for i, match := range matches {
			filtered[i] = m.allFiles[match.Index]
		}
		m.filtered = filtered
	}
	m.cursor = 0
	m.offset = 0
}

func (m *AddFileModel) moveDown() {
	if m.cursor < len(m.filtered)-1 {
		m.cursor++
		m.snapOffset()
	}
}

func (m *AddFileModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
		m.snapOffset()
	}
}

func (m *AddFileModel) pageDown() {
	perPage := m.rows() * m.columns
	m.cursor = min(m.cursor+perPage, max(0, len(m.filtered)-1))
	m.snapOffset()
}

func (m *AddFileModel) pageUp() {
	perPage := m.rows() * m.columns
	m.cursor = max(m.cursor-perPage, 0)
	m.snapOffset()
}

func (m *AddFileModel) goToTop() {
	m.cursor = 0
	m.snapOffset()
}

func (m *AddFileModel) goToBottom() {
	if len(m.filtered) > 0 {
		m.cursor = len(m.filtered) - 1
		m.snapOffset()
	}
}

// snapOffset sets the page-based scroll offset so the cursor is on-screen.
func (m *AddFileModel) snapOffset() {
	perPage := m.rows() * m.columns
	if perPage <= 0 {
		return
	}
	page := m.cursor / perPage
	m.offset = page * perPage
}

// rows returns the number of visible item rows (excludes title, status, pagination).
func (m AddFileModel) rows() int {
	return max(1, m.height-3)
}

func (m AddFileModel) colWidth() int {
	if m.columns <= 1 {
		return m.width
	}
	return (m.width - 2) / 2
}

func (m AddFileModel) totalPages() int {
	perPage := m.rows() * m.columns
	if perPage <= 0 || len(m.filtered) == 0 {
		return 1
	}
	return (len(m.filtered) + perPage - 1) / perPage
}

func (m AddFileModel) currentPage() int {
	perPage := m.rows() * m.columns
	if perPage <= 0 {
		return 0
	}
	return m.cursor / perPage
}

func (m AddFileModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(PaneTitle.Render("Add File to Chezmoi"))
	b.WriteByte('\n')

	// Status — shows filtered count when filter is active
	total := len(m.allFiles)
	visible := len(m.filtered)
	muted := lipgloss.NewStyle().Foreground(MutedColor)
	if visible != total {
		b.WriteString(muted.Render(fmt.Sprintf("%d of %d unmanaged files", visible, total)))
	} else {
		b.WriteString(muted.Render(fmt.Sprintf("%d unmanaged files", total)))
	}
	b.WriteByte('\n')

	if len(m.filtered) == 0 {
		b.WriteString(muted.Render("No matching files"))
		b.WriteString("\n\n")
		b.WriteString(m.filter.View())
		return b.String()
	}

	// Items — column-first fill (down column 1, then column 2)
	rows := m.rows()
	cw := m.colWidth()

	for r := 0; r < rows; r++ {
		for c := 0; c < m.columns; c++ {
			idx := m.offset + c*rows + r
			if c > 0 {
				b.WriteString("  ")
			}
			if idx < len(m.filtered) {
				b.WriteString(renderFileItem(m.filtered[idx], idx == m.cursor, cw))
			} else {
				b.WriteString(strings.Repeat(" ", cw))
			}
		}
		b.WriteByte('\n')
	}

	// Pagination dots
	if tp := m.totalPages(); tp > 1 {
		b.WriteString("  " + renderPaginationDots(m.currentPage(), tp))
	}

	// Gap + filter
	b.WriteString("\n\n")
	b.WriteString(m.filter.View())

	return b.String()
}

func renderFileItem(path string, selected bool, colWidth int) string {
	maxPath := max(0, colWidth-2) // 2 for indicator + space
	display := path
	if len(display) > maxPath && maxPath > 1 {
		display = display[:maxPath-1] + "…"
	}

	// Pad to fill column width so the highlight spans the full row
	text := display
	if pad := maxPath - len(display); pad > 0 {
		text += strings.Repeat(" ", pad)
	}

	if selected {
		return SelectedItem.Render("  " + text)
	}
	return "  " + text
}

func renderPaginationDots(current, total int) string {
	maxDots := min(total, 20)
	activeDot := lipgloss.NewStyle().Foreground(ActiveBorderColor)
	inactiveDot := lipgloss.NewStyle().Foreground(MutedColor)

	var dots []string
	for i := 0; i < maxDots; i++ {
		if i == current {
			dots = append(dots, activeDot.Render("●"))
		} else {
			dots = append(dots, inactiveDot.Render("●"))
		}
	}

	s := strings.Join(dots, " ")
	if total > maxDots {
		s += inactiveDot.Render(" …")
	}
	return s
}
