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
	selected map[string]bool
	cursor   int
	offset   int
	width    int
	height   int
	columns  int
}

func NewAddFileModel(files []string, width, height int) AddFileModel {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(MutedColor)
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
		selected: make(map[string]bool),
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

// SelectedPaths returns all multi-selected file paths. If none are selected,
// returns the currently focused file as a single-element slice (fallback).
func (m AddFileModel) SelectedPaths() []string {
	var paths []string
	for _, f := range m.allFiles {
		if m.selected[f] {
			paths = append(paths, f)
		}
	}
	if len(paths) == 0 {
		if p := m.SelectedPath(); p != "" {
			return []string{p}
		}
	}
	return paths
}

func (m AddFileModel) SelectionCount() int {
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	return count
}

func (m *AddFileModel) ToggleSelected() {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return
	}
	path := m.filtered[m.cursor]
	if m.selected[path] {
		delete(m.selected, path)
	} else {
		m.selected[path] = true
	}
}

// RemoveFiles removes the given paths from the file lists and clears their
// selection state. Keeps the cursor in bounds.
func (m *AddFileModel) RemoveFiles(paths []string) {
	remove := make(map[string]bool, len(paths))
	for _, p := range paths {
		remove[p] = true
		delete(m.selected, p)
	}

	filtered := m.allFiles[:0:0]
	for _, f := range m.allFiles {
		if !remove[f] {
			filtered = append(filtered, f)
		}
	}
	m.allFiles = filtered
	m.applyFilter()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	m.snapOffset()
}

func (m AddFileModel) Update(msg tea.Msg) (AddFileModel, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.Type {
		case tea.KeySpace:
			m.ToggleSelected()
			return m, nil
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

	// Status — shows filtered count when filter is active, plus selection count
	total := len(m.allFiles)
	visible := len(m.filtered)
	sel := m.SelectionCount()
	muted := lipgloss.NewStyle().Foreground(MutedColor)
	var status string
	if visible != total {
		status = fmt.Sprintf("%d of %d unmanaged files", visible, total)
	} else {
		status = fmt.Sprintf("%d unmanaged files", total)
	}
	if sel > 0 {
		status += fmt.Sprintf(" · %d selected", sel)
	}
	b.WriteString(muted.Render(status))
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
				checked := m.selected[m.filtered[idx]]
				b.WriteString(renderFileItem(m.filtered[idx], idx == m.cursor, checked, cw))
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

func renderFileItem(path string, focused, checked bool, colWidth int) string {
	indicator := "  "
	if checked {
		indicator = "✓ "
	}
	maxPath := max(0, colWidth-2) // 2 for indicator
	display := path
	if len(display) > maxPath && maxPath > 1 {
		display = display[:maxPath-1] + "…"
	}

	// Pad to fill column width so the highlight spans the full row
	text := indicator + display
	// indicator is always 2 visual cells wide ("  " or "✓ ")
	if pad := colWidth - 2 - len(display); pad > 0 {
		text += strings.Repeat(" ", pad)
	}

	if focused {
		return SelectedItem.Render(text)
	}
	if checked {
		return lipgloss.NewStyle().Foreground(SuccessColor).Render(text)
	}
	return text
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
