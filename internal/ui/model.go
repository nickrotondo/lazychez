package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

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
	OverlayConfirmGitDiscard
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
	overlay          OverlayMode
	commitInput      textinput.Model
	discardPath      string
	discardUntracked bool

	// Status bar
	statusMsg   string
	statusError bool

	// Cached data for merging
	managedFiles []chezmoi.ManagedFile
	statusData   []chezmoi.StatusEntry

	// Diff cache: home-relative path → diff output (loaded once, refreshed on operations)
	diffCache map[string]string

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
		diffCache:   make(map[string]string),
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
			m.diffCache[msg.Path] = msg.Diff
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

	case PullResultMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Pull failed: %v", msg.Err), true)
			return m, clearStatusAfter()
		}
		m.setStatus("Pulled", false)
		return m, tea.Batch(clearStatusAfter(), m.refreshAll())

	case GitDiscardResultMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Discard failed: %v", msg.Err), true)
			return m, clearStatusAfter()
		}
		m.setStatus(fmt.Sprintf("Discarded %s", msg.Path), false)
		return m, tea.Batch(clearStatusAfter(), m.refreshAll())

	case EditorFinishedMsg:
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Editor error: %v", msg.Err), true)
		} else {
			m.setStatus("Done", false)
		}
		return m, tea.Batch(clearStatusAfter(), m.refreshAll())

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

	case OverlayConfirmGitDiscard:
		switch msg.String() {
		case "y":
			m.overlay = OverlayNone
			if m.discardUntracked {
				return m, cleanFile(m.git, m.discardPath)
			}
			return m, restoreFile(m.git, m.discardPath)
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
		m.setStatus("Waiting for edit...", false)
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
			m.setStatus("Waiting for edit...", false)
			return m, chezmoiEdit(filepath.Join(homeDir, path))
		}
	case "E":
		path := m.fileList.SelectedPath()
		if path != "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err), true)
				return m, clearStatusAfter()
			}
			m.setStatus("Waiting for edit...", false)
			return m, openInEditor(filepath.Join(homeDir, path))
		}
	case "D":
		if sel := m.fileList.SelectedItem(); sel != nil && sel.HasDrift() {
			switch sel.Drift {
			case DriftSourceEdited:
				return m, addFile(m.chezmoi, sel.Path)
			case DriftDestEdited:
				return m, applyFile(m.chezmoi, sel.Path)
			}
		}
	}

	if newPath := m.fileList.SelectedPath(); newPath != prevPath && newPath != "" {
		if diff, ok := m.diffCache[newPath]; ok {
			m.diffView.SetContent(newPath, diff)
			return m, nil
		}
		sel := m.fileList.SelectedItem()
		reverse := sel != nil && sel.Drift == DriftDestEdited
		return m, fetchDiff(m.chezmoi, newPath, reverse)
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
		m.setStatus("Pulling...", false)
		return m, pullFromRemote(m.git)
	case "P":
		m.setStatus("Pushing...", false)
		return m, pushToRemote(m.git)
	case " ":
		if entry, ok := m.gitStatus.SelectedEntry(); ok {
			if entry.XY[0] != ' ' && entry.XY[0] != '?' && entry.XY[1] == ' ' {
				return m, unstageFile(m.git, entry.Path)
			}
			return m, stageFile(m.git, entry.Path)
		}
	case "a":
		return m, stageAllFiles(m.git)
	case "D":
		if entry, ok := m.gitStatus.SelectedEntry(); ok {
			m.discardPath = entry.Path
			m.discardUntracked = entry.XY == "??"
			m.overlay = OverlayConfirmGitDiscard
		}
		return m, nil
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

// --- Internal helpers ---

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

func (m *Model) fetchDiffForSelected() tea.Cmd {
	path := m.fileList.SelectedPath()
	if path == "" {
		return nil
	}
	if diff, ok := m.diffCache[path]; ok {
		m.diffView.SetContent(path, diff)
		return nil
	}
	sel := m.fileList.SelectedItem()
	reverse := sel != nil && sel.Drift == DriftDestEdited
	return fetchDiff(m.chezmoi, path, reverse)
}

func (m Model) fetchGitDiffForSelected() tea.Cmd {
	path := m.gitStatus.SelectedPath()
	if path == "" {
		return nil
	}
	return fetchGitDiff(m.git, path)
}

func (m *Model) fetchDiffForFocusedPane() tea.Cmd {
	switch m.focused {
	case PaneFileList:
		return m.fetchDiffForSelected()
	case PaneGitStatus:
		return m.fetchGitDiffForSelected()
	default:
		return nil
	}
}

func (m *Model) refreshAll() tea.Cmd {
	m.diffCache = make(map[string]string)
	return tea.Batch(
		fetchManagedFiles(m.chezmoi),
		fetchStatus(m.chezmoi),
		fetchGitStatus(m.git),
	)
}
