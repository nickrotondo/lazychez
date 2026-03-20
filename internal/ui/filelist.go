package ui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/nickrotondo/lazychez/internal/chezmoi"
)

type FilterMode int

const (
	FilterInactive FilterMode = iota
	FilterTyping
	FilterLocked
)

type FileItem struct {
	Path          string
	SourceRelPath string
	AddCol        rune // what `chezmoi add` would change: 'M', 'A', 'D', ' '
	ApplyCol      rune // what `chezmoi apply` would change: 'M', 'A', 'D', ' '
	IsHeading     bool
	HeadingText   string

	// Tree view fields
	IsDir        bool   // true for directory nodes
	DirPath      string // for dirs: collapse key
	TreeDepth    int    // indentation level
	TreeName     string // segment name to display
	DirCollapsed bool   // whether this dir is collapsed
}

func (f FileItem) IsDirty() bool {
	return !f.IsHeading && !f.IsDir && (f.AddCol != ' ' || f.ApplyCol != ' ')
}

func (f FileItem) IsTemplate() bool {
	return strings.HasSuffix(f.SourceRelPath, ".tmpl")
}

// StatusCode returns the two-character chezmoi status code for display.
func (f FileItem) StatusCode() string {
	return string(f.AddCol) + string(f.ApplyCol)
}

type FileListModel struct {
	files    []FileItem
	allItems []FileItem // raw items without headings, for filtering
	cursor   int
	offset   int // scroll offset for viewport
	focused  bool
	width    int
	height   int

	// Tree view state
	collapsed map[string]bool // dir path → collapsed, persists across refreshes

	// Filter state
	filterMode  FilterMode
	filterInput textinput.Model
	savedCursor int // cursor position before filter was activated
}

func NewFileListModel() FileListModel {
	return FileListModel{
		collapsed: make(map[string]bool),
	}
}

func (m *FileListModel) SetFiles(files []FileItem) {
	m.allItems = files
	if m.filterMode != FilterInactive {
		// Preserve active filter — reapply it to the new data
		m.applyFilter()
		return
	}
	m.rebuildDisplayList()
	// Clamp cursor if file list shrunk
	if m.cursor >= len(m.files) {
		m.cursor = max(0, len(m.files)-1)
	}
	m.snapCursorToFile()
	m.clampOffset()
}

func (m *FileListModel) rebuildDisplayList() {
	// Partition into dirty (any non-space status) and clean files.
	var dirty, clean []FileItem
	for _, f := range m.allItems {
		if f.IsDirty() {
			dirty = append(dirty, f)
		} else {
			clean = append(clean, f)
		}
	}

	var result []FileItem

	if len(dirty) > 0 {
		tree := buildTree(dirty)
		result = append(result, flattenTree(tree, m.collapsed)...)
	}

	if len(dirty) > 0 && len(clean) > 0 {
		result = append(result, FileItem{IsHeading: true})
	}

	if len(clean) > 0 {
		tree := buildTree(clean)
		result = append(result, flattenTree(tree, m.collapsed)...)
	}

	m.files = result
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
	if len(m.files) == 0 || m.files[m.cursor].IsHeading || m.files[m.cursor].IsDir {
		return ""
	}
	return m.files[m.cursor].Path
}

func (m FileListModel) SelectedItem() *FileItem {
	if len(m.files) == 0 || m.cursor >= len(m.files) || m.files[m.cursor].IsHeading || m.files[m.cursor].IsDir {
		return nil
	}
	return &m.files[m.cursor]
}

func (m FileListModel) SelectedIsDir() bool {
	if len(m.files) == 0 || m.cursor >= len(m.files) {
		return false
	}
	return m.files[m.cursor].IsDir
}

func (m *FileListModel) ToggleCollapse() {
	if len(m.files) == 0 || m.cursor >= len(m.files) || !m.files[m.cursor].IsDir {
		return
	}
	dirPath := m.files[m.cursor].DirPath
	m.collapsed[dirPath] = !m.collapsed[dirPath]
	m.rebuildDisplayList()
	// Clamp cursor
	if m.cursor >= len(m.files) {
		m.cursor = max(0, len(m.files)-1)
	}
	m.clampOffset()
}

func (m FileListModel) FileCount() int {
	count := 0
	for _, f := range m.files {
		if !f.IsHeading && !f.IsDir {
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

// CursorPosition returns the 1-based index among navigable items (excluding headings) and the total count.
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

func (m FileListModel) DirtyCount() int {
	count := 0
	for _, f := range m.files {
		if f.IsDirty() {
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
			lines = append(lines, renderDividerLine(m.width))
			continue
		}

		selected := i == m.cursor && m.focused && m.filterMode != FilterTyping
		indent := strings.Repeat("   ", f.TreeDepth)

		if f.IsDir {
			lines = append(lines, m.renderDirLine(f, indent, selected))
			continue
		}

		lines = append(lines, m.renderFileLine(f, indent, selected))
	}

	return strings.Join(lines, "\n")
}

func (m FileListModel) renderDirLine(f FileItem, indent string, selected bool) string {
	arrow := "▼"
	if f.DirCollapsed {
		arrow = "▶"
	}

	dirStyle := lipgloss.NewStyle().Foreground(DirColor)
	if selected {
		dirStyle = dirStyle.Background(SelectedBg).Bold(true)
	}

	prefix := indent + dirStyle.Render(arrow)
	name := dirStyle.Render(f.TreeName)
	line := prefix + name
	lineWidth := lipgloss.Width(line)
	if pad := m.width - lineWidth; pad > 0 {
		line += dirStyle.Render(strings.Repeat(" ", pad))
	}
	return line
}

func (m FileListModel) renderFileLine(f FileItem, indent string, selected bool) string {
	// Use TreeName (filename segment) for tree view, fall back to full Path
	displayName := f.TreeName
	if displayName == "" {
		displayName = f.Path
	}

	isTemplate := f.IsTemplate()
	tmplSuffix := ""
	tmplWidth := 0
	if isTemplate {
		tmplSuffix = renderTemplateSuffix(selected)
		tmplWidth = 5 // len(".tmpl")
	}

	isDirty := f.AddCol != ' ' || f.ApplyCol != ' '

	if isDirty {
		// Render two-character status code with per-character coloring
		addStr := renderStatusChar(f.AddCol, selected)
		applyStr := renderStatusChar(f.ApplyCol, selected)
		code := addStr + applyStr

		// Plain text width: indent + 2 status chars + space + name + optional .tmpl
		plainWidth := lipgloss.Width(indent) + 3 + lipgloss.Width(displayName) + tmplWidth
		padding := ""
		if pad := m.width - plainWidth; pad > 0 {
			padding = strings.Repeat(" ", pad)
		}

		if selected {
			return SelectedItem.Render(indent) + code + SelectedItem.Render(" "+displayName) + tmplSuffix + SelectedItem.Render(padding)
		}
		return indent + code + " " + displayName + tmplSuffix + padding
	}

	// Clean file — no status code prefix
	plainWidth := lipgloss.Width(indent) + lipgloss.Width(displayName) + tmplWidth
	padding := ""
	if pad := m.width - plainWidth; pad > 0 {
		padding = strings.Repeat(" ", pad)
	}

	if selected {
		return SelectedItem.Render(indent+displayName) + tmplSuffix + SelectedItem.Render(padding)
	}
	return indent + displayName + tmplSuffix + padding
}

// renderTemplateSuffix renders the ".tmpl" indicator in teal.
func renderTemplateSuffix(selected bool) string {
	s := lipgloss.NewStyle().Foreground(TemplateColor)
	if selected {
		s = s.Background(SelectedBg)
	}
	return s.Render(".tmpl")
}

// renderStatusChar colors a single status character based on its value.
func renderStatusChar(ch rune, selected bool) string {
	var color lipgloss.Color
	switch ch {
	case 'M':
		color = ModifiedColor
	case 'A':
		color = AddedColor
	case 'D':
		color = DeletedColor
	default:
		if selected {
			return SelectedItem.Render(" ")
		}
		return " "
	}
	s := lipgloss.NewStyle().Foreground(color).Bold(true)
	if selected {
		s = s.Background(SelectedBg)
	}
	return s.Render(string(ch))
}

// MergeFilesWithStatus builds the merged and sorted file list.
// Dirty files (any non-space status) sort to top by status code then alphabetically.
// Clean files sort alphabetically below.
func MergeFilesWithStatus(managed []chezmoi.ManagedFile, status []chezmoi.StatusEntry) []FileItem {
	statusMap := make(map[string]chezmoi.StatusEntry, len(status))
	for _, s := range status {
		statusMap[s.Path] = s
	}

	items := make([]FileItem, 0, len(managed))
	for _, f := range managed {
		item := FileItem{Path: f.Path, SourceRelPath: f.SourceRelPath, AddCol: ' ', ApplyCol: ' '}
		if s, ok := statusMap[f.Path]; ok {
			item.AddCol = s.AddCol
			item.ApplyCol = s.ApplyCol
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
				Path:     s.Path,
				AddCol:   s.AddCol,
				ApplyCol: s.ApplyCol,
			})
		}
	}

	// Sort: dirty first by status code then alphabetically; clean files alphabetically
	sort.SliceStable(items, func(i, j int) bool {
		di := items[i].AddCol != ' ' || items[i].ApplyCol != ' '
		dj := items[j].AddCol != ' ' || items[j].ApplyCol != ' '
		if di != dj {
			return di // dirty before clean
		}
		if di && dj {
			ci := string(items[i].AddCol) + string(items[i].ApplyCol)
			cj := string(items[j].AddCol) + string(items[j].ApplyCol)
			if ci != cj {
				return ci < cj
			}
		}
		return items[i].Path < items[j].Path
	})

	return items
}

// --- Filter methods ---

func (m FileListModel) IsFiltering() bool {
	return m.filterMode == FilterTyping
}

func (m FileListModel) IsFilterLocked() bool {
	return m.filterMode == FilterLocked
}

func (m *FileListModel) LockFilter() bool {
	// Only lock if there are actual file matches
	if m.FileCount() == 0 {
		return false
	}
	m.filterMode = FilterLocked
	m.filterInput.Blur()
	m.cursor = 0
	m.snapCursorToFile()
	m.offset = 0
	m.clampOffset()
	return true
}

func (m FileListModel) FilterHasMatches() bool {
	return m.FileCount() > 0
}

func (m FileListModel) FilterQuery() string {
	return m.filterInput.Value()
}

func (m *FileListModel) StartFilter() {
	m.filterMode = FilterTyping
	m.savedCursor = m.cursor
	m.filterInput = textinput.New()
	m.filterInput.Prompt = "/ "
	m.filterInput.PromptStyle = HelpKey
	m.filterInput.Focus()
}

func (m *FileListModel) CancelFilter() {
	m.filterMode = FilterInactive
	m.filterInput.Blur()
	m.rebuildDisplayList()
	m.cursor = m.savedCursor
	if m.cursor >= len(m.files) {
		m.cursor = max(0, len(m.files)-1)
	}
	m.clampOffset()
}

func (m *FileListModel) applyFilter() {
	query := m.filterInput.Value()
	if query == "" {
		m.rebuildDisplayList()
		m.filterInput.TextStyle = lipgloss.NewStyle()
		m.cursor = 0
		m.clampOffset()
		return
	}

	paths := make([]string, len(m.allItems))
	for i, f := range m.allItems {
		paths[i] = f.Path
	}

	matches := fuzzy.Find(query, paths)
	filtered := make([]FileItem, len(matches))
	for i, match := range matches {
		filtered[i] = m.allItems[match.Index]
	}

	// Filter shows flat list with full paths (no tree), with divider between dirty/clean
	m.files = insertDivider(filtered)
	m.cursor = 0
	m.snapCursorToFile()
	m.offset = 0
	m.clampOffset()

	// Style the input text red when no files match
	if len(filtered) == 0 {
		m.filterInput.TextStyle = lipgloss.NewStyle().Foreground(ErrorColor)
	} else {
		m.filterInput.TextStyle = lipgloss.NewStyle()
	}
}

// insertDivider adds a divider between dirty and clean files in a flat list.
func insertDivider(files []FileItem) []FileItem {
	hasDirty := false
	hasClean := false
	for _, f := range files {
		if f.IsDirty() {
			hasDirty = true
		} else {
			hasClean = true
		}
		if hasDirty && hasClean {
			break
		}
	}
	if !hasDirty || !hasClean {
		return files
	}

	result := make([]FileItem, 0, len(files)+1)
	inDirty := true
	for _, f := range files {
		if inDirty && !f.IsDirty() {
			result = append(result, FileItem{IsHeading: true})
			inDirty = false
		}
		result = append(result, f)
	}
	return result
}

// renderDividerLine renders a simple horizontal divider line.
func renderDividerLine(width int) string {
	return lipgloss.NewStyle().Foreground(MutedColor).Render(strings.Repeat("─", width))
}
