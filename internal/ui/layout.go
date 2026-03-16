package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// paneChrome is the height consumed by borders (2). Title is embedded in the top border.
const paneChrome = 2

// View renders the full TUI screen.
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
	case OverlayConfirmGitDiscard:
		screen = m.renderOverlay(screen, m.renderConfirmGitDiscard())
	}

	return screen
}

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
		sel := m.fileList.SelectedItem()
		switch {
		case sel != nil && sel.Drift == DriftDestEdited:
			paneHints = []string{
				hint("space", "add (dest → source)"), hint("a", "apply"),
				hint("A", "apply all"), hint("D", "discard"), hint("e/E", "edit"),
			}
		case sel != nil && sel.Drift == DriftSourceEdited:
			paneHints = []string{
				hint("a", "apply (source → dest)"), hint("space", "add"),
				hint("A", "apply all"), hint("D", "discard"), hint("e/E", "edit"),
			}
		default:
			paneHints = []string{
				hint("space", "add"), hint("a", "apply"), hint("A", "apply all"),
				hint("D", "discard"), hint("e/E", "edit"),
			}
		}
		paneHints = append(paneHints, hint("0-2", "panels"))
	case PaneGitStatus:
		paneHints = []string{
			hint("space", "stage"), hint("a", "stage all"),
			hint("c", "commit"), hint("p", "pull"), hint("P", "push"), hint("D", "discard"), hint("0-2", "panels"),
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
    space       Add file (dest → source)
    a           Apply file (source → dest)
    A           Apply all files
    D           Discard drift (revert change)
    e           Edit source (chezmoi edit)
    E           Edit destination file

  Git Actions
    space       Stage file (git add)
    a           Stage all files
    c           Commit (opens input)
    p           Pull from remote
    P           Push to remote
    D           Discard changes

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

func (m Model) renderConfirmGitDiscard() string {
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		PaneTitle.Render("Confirm Discard"),
		fmt.Sprintf("Discard changes to %s?", m.discardPath),
		HelpKey.Render("y")+" yes  "+HelpKey.Render("n")+" no",
	)
	return OverlayStyle.Render(content)
}

// hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence.
func hyperlink(url, text string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}
