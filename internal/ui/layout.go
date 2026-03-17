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
	case OverlayConfirmForget:
		screen = m.renderOverlay(screen, m.renderConfirmForget())
	case OverlayAddFile:
		screen = m.renderOverlay(screen, m.renderAddFileOverlay())
	}

	return screen
}

// viewWide renders the side-by-side layout (>= 100 cols).
// Left column panes size to content; diff takes full right side.
func (m Model) viewWide(fileTitle, gitTitle, diffTitle string) string {
	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth
	contentHeight := m.height - 2

	fileH, gitH := m.distributeLeftColumn(contentHeight)

	fileCur, fileTotal := m.fileList.CursorPosition()
	fileOff, fileTotalLines := m.fileList.ScrollState()
	gitCur, gitTotal := m.gitStatus.CursorPosition()
	gitOff, gitTotalLines := m.gitStatus.ScrollState()
	diffOff, diffTotalLines := m.diffView.ScrollState()

	filePane := m.renderPane(fileTitle, m.fileList.View(), leftWidth, fileH, m.focused == PaneFileList, paneOpts{info: posInfo(fileCur, fileTotal), scrollOff: fileOff, totalLines: fileTotalLines})
	gitPane := m.renderPane(gitTitle, m.gitStatus.View(), leftWidth, gitH, m.focused == PaneGitStatus, paneOpts{info: posInfo(gitCur, gitTotal), scrollOff: gitOff, totalLines: gitTotalLines})
	diffPane := m.renderPane(diffTitle, m.diffView.View(), rightWidth, contentHeight, m.focused == PaneDiff, paneOpts{scrollOff: diffOff, totalLines: diffTotalLines})

	left := lipgloss.JoinVertical(lipgloss.Left, filePane, gitPane)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, diffPane)
}

// viewNarrow renders the stacked layout (< 100 cols).
// Each pane sizes to content; the focused pane absorbs surplus.
func (m Model) viewNarrow(fileTitle, gitTitle, diffTitle string) string {
	w := m.width
	contentHeight := m.height - 2

	fileH, gitH, diffH := m.distributeNarrow(contentHeight)

	fileCur, fileTotal := m.fileList.CursorPosition()
	fileOff, fileTotalLines := m.fileList.ScrollState()
	gitCur, gitTotal := m.gitStatus.CursorPosition()
	gitOff, gitTotalLines := m.gitStatus.ScrollState()
	diffOff, diffTotalLines := m.diffView.ScrollState()

	filePane := m.renderPane(fileTitle, m.fileList.View(), w, fileH, m.focused == PaneFileList, paneOpts{info: posInfo(fileCur, fileTotal), scrollOff: fileOff, totalLines: fileTotalLines})
	gitPane := m.renderPane(gitTitle, m.gitStatus.View(), w, gitH, m.focused == PaneGitStatus, paneOpts{info: posInfo(gitCur, gitTotal), scrollOff: gitOff, totalLines: gitTotalLines})
	diffPane := m.renderPane(diffTitle, m.diffView.View(), w, diffH, m.focused == PaneDiff, paneOpts{scrollOff: diffOff, totalLines: diffTotalLines})

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

type paneOpts struct {
	info       string
	scrollOff  int // current scroll offset (0-based top line)
	totalLines int // total content lines (0 = no scrollbar)
}

func (m Model) renderPane(title, content string, width, height int, active bool, opts ...paneOpts) string {
	var po paneOpts
	if len(opts) > 0 {
		po = opts[0]
	}
	borderColor := InactiveBorderColor
	if active {
		borderColor = ActiveBorderColor
	}

	innerWidth := max(0, width-2)
	innerHeight := max(0, height-2)

	bc := lipgloss.NewStyle().Foreground(borderColor)
	side := bc.Render("│")

	// Top border with title embedded (lazygit style): ╭─[1] Title────╮
	// Color the [N] hotkey only when the pane is focused.
	var titleStr string
	dash := bc.Render("─")
	if idx := strings.Index(title, "] "); idx >= 0 {
		hotkey := title[:idx+1]
		rest := title[idx+2:] // skip the space after "]"
		if active {
			titleStr = PaneTitle.Render(hotkey) + dash + PaneTitle.Render(rest)
		} else {
			titleStr = lipgloss.NewStyle().Bold(true).Render(hotkey) + dash + PaneTitle.Render(rest)
		}
	} else {
		titleStr = PaneTitle.Render(title)
	}
	titleWidth := lipgloss.Width(titleStr)
	// Truncate title if it exceeds available space (innerWidth - 1 for closing corner)
	maxTitle := innerWidth - 1
	if titleWidth > maxTitle && maxTitle > 0 {
		titleStr = lipgloss.NewStyle().MaxWidth(maxTitle).Render(titleStr)
		titleWidth = lipgloss.Width(titleStr)
	}
	pad := max(0, innerWidth-titleWidth-1)
	topLine := bc.Render("╭─") + titleStr + bc.Render(strings.Repeat("─", pad)+"╮")

	// Render content to exact dimensions
	body := lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		Render(content)

	// Compute scrollbar thumb range (lazygit style: colored block on right border).
	thumbStart, thumbEnd := -1, -1
	if po.totalLines > innerHeight && innerHeight > 0 {
		thumbSize := max(1, innerHeight*innerHeight/po.totalLines)
		maxOff := po.totalLines - innerHeight
		thumbStart = po.scrollOff * (innerHeight - thumbSize) / maxOff
		thumbEnd = thumbStart + thumbSize
	}
	thumbStyle := lipgloss.NewStyle().Foreground(borderColor)
	thumb := thumbStyle.Render("▐")

	lines := strings.Split(body, "\n")
	var mid strings.Builder
	for i := 0; i < innerHeight; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		rightBorder := side
		if i >= thumbStart && i < thumbEnd {
			rightBorder = thumb
		}
		mid.WriteString(side + line + rightBorder + "\n")
	}

	// Bottom border with optional position info (lazygit style): ╰────1 of 12─╯
	var bottomLine string
	if po.info != "" {
		infoStyle := lipgloss.NewStyle().Bold(true)
		if active {
			infoStyle = PaneTitle
		}
		infoStr := infoStyle.Render(po.info)
		infoWidth := lipgloss.Width(infoStr)
		fill := max(0, innerWidth-infoWidth-1)
		bottomLine = bc.Render("╰"+strings.Repeat("─", fill)) + infoStr + bc.Render("─╯")
	} else {
		bottomLine = bc.Render("╰" + strings.Repeat("─", innerWidth) + "╯")
	}

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
	return style.Width(m.width).Render(m.statusMsg)
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
				hint("A", "apply all"), hint("D", "discard"), hint("+", "new"), hint("x", "forget"), hint("e/E", "edit"),
			}
		case sel != nil && sel.Drift == DriftSourceEdited:
			paneHints = []string{
				hint("a", "apply (source → dest)"), hint("space", "add"),
				hint("A", "apply all"), hint("D", "discard"), hint("+", "new"), hint("x", "forget"), hint("e/E", "edit"),
			}
		default:
			paneHints = []string{
				hint("space", "add"), hint("a", "apply"), hint("A", "apply all"),
				hint("D", "discard"), hint("+", "new"), hint("x", "forget"), hint("e/E", "edit"),
			}
		}
		paneHints = append(paneHints, hint("0-2", "panels"))
	case PaneGitStatus:
		paneHints = []string{
			hint("space", "stage"), hint("a", "stage all"),
			hint("c", "commit"), hint("p", "pull"), hint("P", "push"), hint("D", "discard"), hint("0-2", "panels"),
		}
	case PaneDiff:
		paneHints = []string{hint("esc", "back"), hint("0-2", "panels")}
	}

	globalHints := []string{
		hint("C", "config"), hint("?", "help"), hint("q", "quit"),
	}

	allHints := append(paneHints, globalHints...)
	right := hyperlink("https://x.com/nicklrotondo", FooterLink.Render("𝕏 @nicklrotondo")) + " "
	rightWidth := lipgloss.Width(right)
	ellipsis := HelpSep.Render(" | ") + HelpDesc.Render("…")

	// Progressively drop hints from the right until they fit.
	left := " " + strings.Join(allHints, sep)
	for len(allHints) > 1 && lipgloss.Width(left)+rightWidth+1 > m.width {
		allHints = allHints[:len(allHints)-1]
		left = " " + strings.Join(allHints, sep) + ellipsis
	}

	gap := m.width - lipgloss.Width(left) - rightWidth
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
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#172b32")),
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
    esc         Back from diff panel

  File Actions
    space       Add file (dest → source)
    a           Apply file (source → dest)
    A           Apply all files
    D           Discard drift (revert change)
    e           Edit source (chezmoi edit)
    E           Edit destination file
    +           Add unmanaged file
    x           Forget file (unmanage)

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

func (m Model) renderConfirmForget() string {
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		PaneTitle.Render("Confirm Forget"),
		fmt.Sprintf("Remove %s from chezmoi management?", m.forgetPath),
		HelpKey.Render("y")+" yes  "+HelpKey.Render("n")+" no",
	)
	return OverlayStyle.Render(content)
}

func (m Model) renderAddFileOverlay() string {
	hint := HelpDesc.Render("type to filter") +
		HelpSep.Render(" · ") +
		HelpKey.Render("enter") + " " + HelpDesc.Render("add") +
		HelpSep.Render(" · ") +
		HelpKey.Render("esc") + " " + HelpDesc.Render("close")
	content := m.addFile.View() + "\n" + hint
	w := min(100, max(40, m.width*80/100))
	return OverlayStyle.Width(w).Render(content)
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

// posInfo formats a "X of Y" string, or empty if total is 0.
func posInfo(current, total int) string {
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("%d of %d", current, total)
}

// hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence.
func hyperlink(url, text string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}
