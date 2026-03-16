package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nickrotondo/lazychez/internal/chezmoi"
	"github.com/nickrotondo/lazychez/internal/git"
)

type PaneID int

const (
	PaneFileList PaneID = iota
	PaneGitStatus
	PaneDiff
	paneCount
)

type OverlayMode int

const (
	OverlayNone OverlayMode = iota
	OverlayHelp
	OverlayCommit
	OverlayConfirmApplyAll
)

// narrowBreakpoint is the width below which we switch to stacked layout.
const narrowBreakpoint = 100

type Model struct {
	width   int
	height  int
	focused PaneID

	fileList  FileListModel
	gitStatus GitStatusModel
	diffView  DiffViewModel

	// Overlay state
	overlay     OverlayMode
	commitInput textinput.Model

	// Status bar
	statusMsg   string
	statusError bool

	// Cached data for merging
	managedFiles []chezmoi.ManagedFile
	statusData   []chezmoi.StatusEntry

	chezmoi chezmoi.Runner
	git     git.Runner
}

func New(chezmoiRunner chezmoi.Runner, gitRunner git.Runner) Model {
	ti := textinput.New()
	ti.Placeholder = "commit message..."
	ti.CharLimit = 120

	return Model{
		focused:     PaneFileList,
		fileList:    NewFileListModel(),
		gitStatus:   NewGitStatusModel(),
		diffView:    NewDiffViewModel(),
		commitInput: ti,
		chezmoi:     chezmoiRunner,
		git:         gitRunner,
	}
}

func (m Model) isNarrow() bool {
	return m.width < narrowBreakpoint
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchManagedFiles(m.chezmoi),
		fetchStatus(m.chezmoi),
		fetchGitStatus(m.git),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateDimensions()
		return m, nil

	case ManagedFilesMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Error: %v", msg.Err), true)
			return m, clearStatusAfter()
		}
		m.managedFiles = msg.Files
		m.rebuildFileList()
		m.updateDimensions()
		return m, m.fetchDiffForSelected()

	case StatusMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Error: %v", msg.Err), true)
			return m, clearStatusAfter()
		}
		m.statusData = msg.Entries
		m.rebuildFileList()
		m.updateDimensions()
		return m, m.fetchDiffForSelected()

	case DiffMsg:
		if msg.Err != nil {
			m.diffView.SetContent(msg.Path, fmt.Sprintf("Error loading diff: %v", msg.Err))
		} else {
			m.diffView.SetContent(msg.Path, msg.Diff)
		}
		return m, nil

	case GitStatusMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Git error: %v", msg.Err), true)
			return m, clearStatusAfter()
		}
		m.gitStatus.SetEntries(msg.Entries)
		m.rebuildFileList()
		m.updateDimensions()
		if m.focused == PaneGitStatus {
			return m, m.fetchGitDiffForSelected()
		}
		return m, nil

	case AddResultMsg:
		if msg.Err != nil {
			var tmplErr *chezmoi.TemplateEditError
			if errors.As(msg.Err, &tmplErr) {
				m.setStatus(fmt.Sprintf("%s is a template — use chezmoi edit", msg.Path), true)
			} else {
				m.setStatus(fmt.Sprintf("Error adding %s: %v", msg.Path, msg.Err), true)
			}
		} else {
			m.setStatus(fmt.Sprintf("Added %s", msg.Path), false)
		}
		return m, tea.Batch(clearStatusAfter(), m.refreshAll())

	case ApplyResultMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Error applying %s: %v", msg.Path, msg.Err), true)
		} else {
			m.setStatus(fmt.Sprintf("Applied %s", msg.Path), false)
		}
		return m, tea.Batch(clearStatusAfter(), m.refreshAll())

	case ApplyAllResultMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Error applying all: %v", msg.Err), true)
		} else {
			m.setStatus("Applied all files", false)
		}
		return m, tea.Batch(clearStatusAfter(), m.refreshAll())

	case GitStageResultMsg:
		if msg.Err != nil {
			if msg.Path != "" {
				m.setStatus(fmt.Sprintf("Error staging %s: %v", msg.Path, msg.Err), true)
			} else {
				m.setStatus(fmt.Sprintf("Error staging all: %v", msg.Err), true)
			}
		} else {
			if msg.Path != "" {
				m.setStatus(fmt.Sprintf("Staged %s", msg.Path), false)
			} else {
				m.setStatus("Staged all files", false)
			}
		}
		return m, tea.Batch(clearStatusAfter(), fetchGitStatus(m.git))

	case CommitResultMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Commit failed: %v", msg.Err), true)
		} else {
			m.setStatus("Committed", false)
		}
		return m, tea.Batch(clearStatusAfter(), fetchGitStatus(m.git))

	case PushResultMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Push failed: %v", msg.Err), true)
		} else {
			m.setStatus("Pushed", false)
		}
		return m, clearStatusAfter()

	case EditorFinishedMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Editor error: %v", msg.Err), true)
			return m, clearStatusAfter()
		}
		return m, m.refreshAll()

	case ClearStatusMsg:
		m.statusMsg = ""
		m.statusError = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward to diff viewport for scroll updates
	if m.focused == PaneDiff {
		var cmd tea.Cmd
		m.diffView, cmd = m.diffView.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Overlay-specific key handling
	switch m.overlay {
	case OverlayHelp:
		switch msg.String() {
		case "?", "esc", "q":
			m.overlay = OverlayNone
		}
		return m, nil

	case OverlayCommit:
		return m.handleCommitKey(msg)

	case OverlayConfirmApplyAll:
		switch msg.String() {
		case "y":
			m.overlay = OverlayNone
			return m, applyAll(m.chezmoi)
		case "n", "esc":
			m.overlay = OverlayNone
		}
		return m, nil
	}

	// Global keys
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		m.focused = (m.focused + 1) % paneCount
		m.updateDimensions()
		return m, m.fetchDiffForFocusedPane()
	case "shift+tab":
		m.focused = (m.focused - 1 + paneCount) % paneCount
		m.updateDimensions()
		return m, m.fetchDiffForFocusedPane()
	case "1":
		m.focused = PaneFileList
		m.updateDimensions()
		return m, m.fetchDiffForFocusedPane()
	case "2":
		m.focused = PaneGitStatus
		m.updateDimensions()
		return m, m.fetchDiffForFocusedPane()
	case "0":
		m.focused = PaneDiff
		m.updateDimensions()
		return m, nil
	case "r":
		return m, m.refreshAll()
	case "?":
		m.overlay = OverlayHelp
		return m, nil
	case "C":
		return m, chezmoiEditConfig()
	}

	// Pane-specific keys
	switch m.focused {
	case PaneFileList:
		return m.handleFileListKey(msg)
	case PaneGitStatus:
		return m.handleGitStatusKey(msg)
	case PaneDiff:
		var cmd tea.Cmd
		m.diffView, cmd = m.diffView.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleFileListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	prevPath := m.fileList.SelectedPath()

	switch msg.String() {
	case "j", "down":
		m.fileList.MoveDown()
	case "k", "up":
		m.fileList.MoveUp()
	case "g", "home":
		m.fileList.GoToTop()
	case "G", "end":
		m.fileList.GoToBottom()
	case " ":
		path := m.fileList.SelectedPath()
		if path != "" {
			return m, addFile(m.chezmoi, path)
		}
	case "a":
		path := m.fileList.SelectedPath()
		if path != "" {
			return m, applyFile(m.chezmoi, path)
		}
	case "A":
		m.overlay = OverlayConfirmApplyAll
		return m, nil
	case "e":
		path := m.fileList.SelectedPath()
		if path != "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err), true)
				return m, clearStatusAfter()
			}
			return m, chezmoiEdit(homeDir + "/" + path)
		}
	case "E":
		path := m.fileList.SelectedPath()
		if path != "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err), true)
				return m, clearStatusAfter()
			}
			return m, openInEditor(homeDir + "/" + path)
		}
	}

	if newPath := m.fileList.SelectedPath(); newPath != prevPath && newPath != "" {
		return m, fetchDiff(m.chezmoi, newPath)
	}

	return m, nil
}

func (m Model) handleGitStatusKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	prevPath := m.gitStatus.SelectedPath()

	switch msg.String() {
	case "j", "down":
		m.gitStatus.MoveDown()
	case "k", "up":
		m.gitStatus.MoveUp()
	case "g", "home":
		m.gitStatus.GoToTop()
	case "G", "end":
		m.gitStatus.GoToBottom()
	case "c":
		m.overlay = OverlayCommit
		m.commitInput.Reset()
		m.commitInput.Focus()
		return m, textinput.Blink
	case "p":
		return m, pushToRemote(m.git)
	case " ":
		path := m.gitStatus.SelectedPath()
		if path != "" {
			return m, stageFile(m.git, path)
		}
	case "a":
		return m, stageAllFiles(m.git)
	}

	if newPath := m.gitStatus.SelectedPath(); newPath != prevPath && newPath != "" {
		return m, fetchGitDiff(m.git, newPath)
	}

	return m, nil
}

func (m Model) handleCommitKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.overlay = OverlayNone
		m.commitInput.Blur()
		return m, nil
	case "enter":
		message := m.commitInput.Value()
		if message == "" {
			return m, nil
		}
		m.overlay = OverlayNone
		m.commitInput.Blur()
		return m, commitChanges(m.git, message)
	}

	var cmd tea.Cmd
	m.commitInput, cmd = m.commitInput.Update(msg)
	return m, cmd
}

// --- View ---

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	fileTitle := fmt.Sprintf("[1] Managed Files (%d)", m.fileList.FileCount())
	if dc := m.fileList.DriftCount(); dc > 0 {
		fileTitle = fmt.Sprintf("[1] Managed Files (%d) · %d drifted", m.fileList.FileCount(), dc)
	}
	gitTitle := fmt.Sprintf("[2] Source Git (%d)", m.gitStatus.EntryCount())
	diffTitle := "[0] Diff"
	if p := m.diffView.Path(); p != "" {
		diffTitle = fmt.Sprintf("[0] Diff — %s", p)
	}

	var main string
	if m.isNarrow() {
		main = m.viewNarrow(fileTitle, gitTitle, diffTitle)
	} else {
		main = m.viewWide(fileTitle, gitTitle, diffTitle)
	}

	statusLine := m.renderStatusBar()
	footer := m.renderFooter()

	screen := lipgloss.JoinVertical(lipgloss.Left, main, statusLine, footer)

	switch m.overlay {
	case OverlayHelp:
		screen = m.renderOverlay(screen, m.renderHelp())
	case OverlayCommit:
		screen = m.renderOverlay(screen, m.renderCommitInput())
	case OverlayConfirmApplyAll:
		screen = m.renderOverlay(screen, m.renderConfirmApplyAll())
	}

	return screen
}

// paneChrome is the height consumed by borders (2). Title is embedded in the top border.
const paneChrome = 2

// viewWide renders the side-by-side layout (>= 100 cols).
// Left column panes size to content; diff takes full right side.
func (m Model) viewWide(fileTitle, gitTitle, diffTitle string) string {
	leftWidth := m.width * 30 / 100
	rightWidth := m.width - leftWidth
	contentHeight := m.height - 2

	fileH, gitH := m.distributeLeftColumn(contentHeight)

	filePane := m.renderPane(fileTitle, m.fileList.View(), leftWidth, fileH, m.focused == PaneFileList)
	gitPane := m.renderPane(gitTitle, m.gitStatus.View(), leftWidth, gitH, m.focused == PaneGitStatus)
	diffPane := m.renderPane(diffTitle, m.diffView.View(), rightWidth, contentHeight, m.focused == PaneDiff)

	left := lipgloss.JoinVertical(lipgloss.Left, filePane, gitPane)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, diffPane)
}

// viewNarrow renders the stacked layout (< 100 cols).
// Each pane sizes to content; the focused pane absorbs surplus.
func (m Model) viewNarrow(fileTitle, gitTitle, diffTitle string) string {
	w := m.width
	contentHeight := m.height - 2

	fileH, gitH, diffH := m.distributeNarrow(contentHeight)

	filePane := m.renderPane(fileTitle, m.fileList.View(), w, fileH, m.focused == PaneFileList)
	gitPane := m.renderPane(gitTitle, m.gitStatus.View(), w, gitH, m.focused == PaneGitStatus)
	diffPane := m.renderPane(diffTitle, m.diffView.View(), w, diffH, m.focused == PaneDiff)

	return lipgloss.JoinVertical(lipgloss.Left, filePane, gitPane, diffPane)
}

// distributeLeftColumn computes heights for the file list and git status
// panes in wide mode. Each pane gets its content height + chrome, capped at
// 70% of available. The focused pane absorbs surplus; if neither left pane
// is focused, surplus goes to file list.
func (m Model) distributeLeftColumn(available int) (fileH, gitH int) {
	maxH := available * 70 / 100

	fileDesired := min(m.fileList.ContentLines()+paneChrome, maxH)
	gitDesired := min(m.gitStatus.ContentLines()+paneChrome, maxH)

	// Ensure minimum
	fileDesired = max(fileDesired, paneChrome)
	gitDesired = max(gitDesired, paneChrome)

	total := fileDesired + gitDesired

	if total <= available {
		// Surplus — give it to the focused left pane, or file list
		surplus := available - total
		switch m.focused {
		case PaneGitStatus:
			gitDesired += surplus
		default:
			fileDesired += surplus
		}
		return fileDesired, gitDesired
	}

	// Over budget — proportionally reduce, respecting minimums
	fileH = max(paneChrome, available*fileDesired/total)
	gitH = available - fileH
	gitH = max(gitH, paneChrome)
	fileH = available - gitH
	return fileH, gitH
}

// distributeNarrow computes heights for all three panes in narrow mode.
// Same content-aware logic: each pane asks for content + chrome, focused
// pane absorbs surplus.
func (m Model) distributeNarrow(available int) (fileH, gitH, diffH int) {
	maxH := available * 60 / 100

	fileDesired := min(m.fileList.ContentLines()+paneChrome, maxH)
	gitDesired := min(m.gitStatus.ContentLines()+paneChrome, maxH)
	// Diff always wants as much as possible
	diffDesired := maxH

	fileDesired = max(fileDesired, paneChrome)
	gitDesired = max(gitDesired, paneChrome)
	diffDesired = max(diffDesired, paneChrome)

	total := fileDesired + gitDesired + diffDesired

	if total <= available {
		surplus := available - total
		switch m.focused {
		case PaneFileList:
			fileDesired += surplus
		case PaneGitStatus:
			gitDesired += surplus
		default:
			diffDesired += surplus
		}
		return fileDesired, gitDesired, diffDesired
	}

	// Over budget — give focused pane its desired, compress others
	focusedH := maxH
	var otherA, otherB *int
	switch m.focused {
	case PaneFileList:
		fileDesired = focusedH
		otherA, otherB = &gitDesired, &diffDesired
	case PaneGitStatus:
		gitDesired = focusedH
		otherA, otherB = &fileDesired, &diffDesired
	default:
		diffDesired = focusedH
		otherA, otherB = &fileDesired, &gitDesired
	}

	remaining := available - focusedH
	aRatio := *otherA
	bRatio := *otherB
	ratioTotal := aRatio + bRatio
	if ratioTotal > 0 {
		*otherA = max(paneChrome, remaining*aRatio/ratioTotal)
		*otherB = max(paneChrome, remaining-*otherA)
	} else {
		*otherA = remaining / 2
		*otherB = remaining - *otherA
	}

	return fileDesired, gitDesired, diffDesired
}

func (m Model) renderPane(title, content string, width, height int, active bool) string {
	borderColor := InactiveBorderColor
	if active {
		borderColor = ActiveBorderColor
	}

	innerWidth := max(0, width-2)
	innerHeight := max(0, height-2)

	bc := lipgloss.NewStyle().Foreground(borderColor)
	side := bc.Render("│")

	// Top border with title embedded (lazygit style): ╭─[1] Title────╮
	titleStr := PaneTitle.Render(title)
	titleWidth := lipgloss.Width(titleStr)
	pad := max(0, innerWidth-titleWidth-1)
	topLine := bc.Render("╭─") + titleStr + bc.Render(strings.Repeat("─", pad)+"╮")

	// Render content to exact dimensions
	body := lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		Render(content)

	// Wrap each content line with side borders
	lines := strings.Split(body, "\n")
	var mid strings.Builder
	for i := 0; i < innerHeight; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		mid.WriteString(side + line + side + "\n")
	}

	// Bottom border
	bottomLine := bc.Render("╰" + strings.Repeat("─", innerWidth) + "╯")

	return topLine + "\n" + mid.String() + bottomLine
}

func (m Model) renderStatusBar() string {
	if m.statusMsg == "" {
		return ""
	}
	style := StatusBarSuccess
	if m.statusError {
		style = StatusBarError
	}
	return " " + style.Render(m.statusMsg)
}

func (m Model) renderFooter() string {
	sep := HelpSep.Render(" | ")
	hint := func(key, desc string) string {
		return HelpKey.Render(key) + " " + HelpDesc.Render(desc)
	}

	var paneHints []string
	switch m.focused {
	case PaneFileList:
		paneHints = []string{
			hint("space", "add"), hint("a", "apply"), hint("A", "apply all"),
			hint("e/E", "edit"), hint("0-2", "panels"),
		}
	case PaneGitStatus:
		paneHints = []string{
			hint("space", "stage"), hint("a", "stage all"),
			hint("c", "commit"), hint("p", "push"), hint("0-2", "panels"),
		}
	case PaneDiff:
		paneHints = []string{hint("0-2", "panels")}
	}

	globalHints := []string{
		hint("C", "config"), hint("?", "help"), hint("q", "quit"),
	}

	left := " " + strings.Join(append(paneHints, globalHints...), sep)
	right := hyperlink("https://x.com/nicklrotondo", FooterLink.Render("𝕏 @nicklrotondo")) + " "

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := m.width - leftWidth - rightWidth
	if gap < 1 {
		return left
	}

	return left + strings.Repeat(" ", gap) + right
}

func (m Model) renderOverlay(background, overlay string) string {
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#1a1b26")),
	)
}

func (m Model) renderHelp() string {
	help := `Keybindings

  Navigation
    j/k         Move up/down
    g/G         Jump to top/bottom
    0/1/2       Jump to panel
    tab         Next panel
    shift+tab   Previous panel

  File Actions
    space       Add file (chezmoi add)
    a           Apply file (chezmoi apply)
    A           Apply all files
    e           Edit source (chezmoi edit)
    E           Edit destination file

  Git Actions
    space       Stage file (git add)
    a           Stage all files
    c           Commit (opens input)
    p           Push to remote

  General
    r           Refresh all panes
    C           Edit chezmoi config
    ?           Toggle this help
    q           Quit`

	return OverlayStyle.Render(help)
}

func (m Model) renderCommitInput() string {
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		PaneTitle.Render("Commit Message"),
		m.commitInput.View(),
		HelpDesc.Render("enter to commit · esc to cancel"),
	)
	return OverlayStyle.Width(50).Render(content)
}

func (m Model) renderConfirmApplyAll() string {
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		PaneTitle.Render("Confirm Apply All"),
		"Apply all managed files to destination?",
		HelpKey.Render("y")+" yes  "+HelpKey.Render("n")+" no",
	)
	return OverlayStyle.Render(content)
}

// --- Internal helpers ---

// hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence.
func hyperlink(url, text string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

func (m *Model) setStatus(msg string, isError bool) {
	m.statusMsg = msg
	m.statusError = isError
}

func (m *Model) updateDimensions() {
	if m.isNarrow() {
		m.updateDimensionsNarrow()
	} else {
		m.updateDimensionsWide()
	}
	m.syncFocus()
}

func (m *Model) updateDimensionsWide() {
	leftWidth := m.width * 30 / 100
	rightWidth := m.width - leftWidth
	contentHeight := m.height - 2

	fileH, gitH := m.distributeLeftColumn(contentHeight)

	m.fileList.SetDimensions(max(0, leftWidth-2), max(0, fileH-paneChrome))
	m.gitStatus.SetDimensions(max(0, leftWidth-2), max(0, gitH-paneChrome))
	m.diffView.SetDimensions(max(0, rightWidth-2), max(0, contentHeight-paneChrome))
}

func (m *Model) updateDimensionsNarrow() {
	innerW := max(0, m.width-2)
	contentHeight := m.height - 2

	fileH, gitH, diffH := m.distributeNarrow(contentHeight)

	m.fileList.SetDimensions(innerW, max(0, fileH-paneChrome))
	m.gitStatus.SetDimensions(innerW, max(0, gitH-paneChrome))
	m.diffView.SetDimensions(innerW, max(0, diffH-paneChrome))
}

func (m *Model) syncFocus() {
	m.fileList.SetFocused(m.focused == PaneFileList)
	m.gitStatus.SetFocused(m.focused == PaneGitStatus)
	m.diffView.SetFocused(m.focused == PaneDiff)
}

func (m *Model) rebuildFileList() {
	gitPaths := make(map[string]bool, len(m.gitStatus.entries))
	for _, e := range m.gitStatus.entries {
		gitPaths[e.Path] = true
	}
	items := MergeFilesWithStatus(m.managedFiles, m.statusData, gitPaths)
	m.fileList.SetFiles(items)
}

func (m Model) fetchDiffForSelected() tea.Cmd {
	path := m.fileList.SelectedPath()
	if path == "" {
		return nil
	}
	return fetchDiff(m.chezmoi, path)
}

func (m Model) fetchGitDiffForSelected() tea.Cmd {
	path := m.gitStatus.SelectedPath()
	if path == "" {
		return nil
	}
	return fetchGitDiff(m.git, path)
}

func (m Model) fetchDiffForFocusedPane() tea.Cmd {
	switch m.focused {
	case PaneFileList:
		return m.fetchDiffForSelected()
	case PaneGitStatus:
		return m.fetchGitDiffForSelected()
	default:
		return nil
	}
}

func (m Model) refreshAll() tea.Cmd {
	return tea.Batch(
		fetchManagedFiles(m.chezmoi),
		fetchStatus(m.chezmoi),
		fetchGitStatus(m.git),
	)
}

// --- tea.Cmd factories ---

func fetchManagedFiles(r chezmoi.Runner) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		files, err := r.Managed(ctx)
		return ManagedFilesMsg{Files: files, Err: err}
	}
}

func fetchStatus(r chezmoi.Runner) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		entries, err := r.Status(ctx)
		return StatusMsg{Entries: entries, Err: err}
	}
}

func fetchDiff(r chezmoi.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		diff, err := r.Diff(ctx, path)
		return DiffMsg{Path: path, Diff: diff, Err: err}
	}
}

func fetchGitDiff(r git.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		diff, err := r.Diff(ctx, path)
		return DiffMsg{Path: path, Diff: diff, Err: err}
	}
}

func addFile(r chezmoi.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := r.Add(ctx, path)
		return AddResultMsg{Path: path, Err: err}
	}
}

func applyFile(r chezmoi.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := r.Apply(ctx, path)
		return ApplyResultMsg{Path: path, Err: err}
	}
}

func applyAll(r chezmoi.Runner) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := r.ApplyAll(ctx)
		return ApplyAllResultMsg{Err: err}
	}
}

func stageFile(r git.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := r.Add(ctx, path)
		return GitStageResultMsg{Path: path, Err: err}
	}
}

func stageAllFiles(r git.Runner) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := r.AddAll(ctx)
		return GitStageResultMsg{Err: err}
	}
}

func fetchGitStatus(r git.Runner) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		entries, err := r.Status(ctx)
		var uiEntries []GitStatusEntry
		for _, e := range entries {
			uiEntries = append(uiEntries, GitStatusEntry{XY: e.XY, Path: e.Path})
		}
		return GitStatusMsg{Entries: uiEntries, Err: err}
	}
}

func commitChanges(r git.Runner, message string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := r.Commit(ctx, message)
		return CommitResultMsg{Err: err}
	}
}

func pushToRemote(r git.Runner) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := r.Push(ctx)
		return PushResultMsg{Err: err}
	}
}

// GUI editors that don't need terminal control — run them async so the TUI stays visible.
var guiEditors = map[string]bool{
	"code": true, "code-insiders": true,
	"cursor": true,
	"subl": true, "sublime_text": true,
	"zed": true,
	"atom": true,
	"fleet": true,
	"idea": true, "goland": true, "webstorm": true, "pycharm": true,
}

func isGUIEditor(command string) bool {
	base := filepath.Base(command)
	return guiEditors[base]
}

func resolveEditor() (string, []string) {
	out, err := exec.Command("chezmoi", "dump-config", "--format=json").Output()
	if err == nil {
		var cfg struct {
			Edit struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			} `json:"edit"`
		}
		if json.Unmarshal(out, &cfg) == nil && cfg.Edit.Command != "" {
			return cfg.Edit.Command, cfg.Edit.Args
		}
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor, nil
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, nil
	}
	return "vi", nil
}

func editorCmd(editor string, args []string, filePath string) tea.Cmd {
	fullArgs := append(args, filePath)
	if isGUIEditor(editor) {
		return func() tea.Msg {
			err := exec.Command(editor, fullArgs...).Run()
			return EditorFinishedMsg{Err: err}
		}
	}
	c := exec.Command(editor, fullArgs...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func openInEditor(filePath string) tea.Cmd {
	editor, args := resolveEditor()
	return editorCmd(editor, args, filePath)
}

func chezmoiEdit(filePath string) tea.Cmd {
	editor, _ := resolveEditor()
	if isGUIEditor(editor) {
		return func() tea.Msg {
			err := exec.Command("chezmoi", "edit", filePath).Run()
			return EditorFinishedMsg{Err: err}
		}
	}
	c := exec.Command("chezmoi", "edit", filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func chezmoiEditConfig() tea.Cmd {
	editor, _ := resolveEditor()
	if isGUIEditor(editor) {
		return func() tea.Msg {
			err := exec.Command("chezmoi", "edit-config").Run()
			return EditorFinishedMsg{Err: err}
		}
	}
	c := exec.Command("chezmoi", "edit-config")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func clearStatusAfter() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}
