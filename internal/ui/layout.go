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

	fileTitle := "[1] Managed Files"
	if dc := m.fileList.DriftCount(); dc > 0 {
		fileTitle = fmt.Sprintf("[1] Managed Files · %d drifted", dc)
	}
	gitTitle := "[2] Source Git"
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
// Unfocused panes render as single-line collapsed bars.
func (m Model) viewNarrow(fileTitle, gitTitle, diffTitle string) string {
	w := m.width
	contentHeight := m.height - 2

	fileH, gitH, diffH := m.distributeNarrow(contentHeight)

	var filePane, gitPane, diffPane string

	fileCur, fileTotal := m.fileList.CursorPosition()
	if fileH == collapsedHeight {
		filePane = renderCollapsedPane(fileTitle, w, posInfo(fileCur, fileTotal))
	} else {
		fileOff, fileTotalLines := m.fileList.ScrollState()
		filePane = m.renderPane(fileTitle, m.fileList.View(), w, fileH, m.focused == PaneFileList, paneOpts{info: posInfo(fileCur, fileTotal), scrollOff: fileOff, totalLines: fileTotalLines})
	}

	gitCur, gitTotal := m.gitStatus.CursorPosition()
	if gitH == collapsedHeight {
		gitPane = renderCollapsedPane(gitTitle, w, posInfo(gitCur, gitTotal))
	} else {
		gitOff, gitTotalLines := m.gitStatus.ScrollState()
		gitPane = m.renderPane(gitTitle, m.gitStatus.View(), w, gitH, m.focused == PaneGitStatus, paneOpts{info: posInfo(gitCur, gitTotal), scrollOff: gitOff, totalLines: gitTotalLines})
	}

	// Diff pane is always rendered as a full pane, never collapsed.
	diffOff, diffTotalLines := m.diffView.ScrollState()
	diffPane = m.renderPane(diffTitle, m.diffView.View(), w, diffH, m.focused == PaneDiff, paneOpts{scrollOff: diffOff, totalLines: diffTotalLines})

	return lipgloss.JoinVertical(lipgloss.Left, filePane, gitPane, diffPane)
}

// renderCollapsedPane renders a single-line bar for an unfocused pane in narrow mode.
// Format: ──[2]─Source Git──────────1 of 5─
func renderCollapsedPane(title string, width int, info string) string {
	bc := lipgloss.NewStyle().Foreground(InactiveBorderColor)
	dash := bc.Render("─")
	inactive := lipgloss.NewStyle().Bold(true)

	var titleStr string
	if idx := strings.Index(title, "] "); idx >= 0 {
		hotkey := title[:idx+1]
		rest := title[idx+2:]
		titleStr = inactive.Render(hotkey) + dash + inactive.Render(rest)
	} else {
		titleStr = inactive.Render(title)
	}

	var infoStr string
	var infoWidth int
	if info != "" {
		infoStr = inactive.Render(info) + bc.Render("──")
		infoWidth = lipgloss.Width(infoStr)
	}

	titleWidth := lipgloss.Width(titleStr)
	pad := max(0, width-titleWidth-infoWidth-2)
	return bc.Render("──") + titleStr + bc.Render(strings.Repeat("─", pad)) + infoStr
}

// distributeLeftColumn splits the left column 50/50 between file list and
// git status panes in wide mode.
func (m Model) distributeLeftColumn(available int) (fileH, gitH int) {
	fileH = available / 2
	gitH = available - fileH
	return fileH, gitH
}

// collapsedHeight is the height of an unfocused pane in narrow mode (single-line bar).
const collapsedHeight = 1

// distributeNarrow computes heights for all three panes in narrow mode.
// The diff pane always stays visible (never collapses). The "active" side
// pane (focused, or prevFocused when diff is focused) stays expanded and
// splits ~50/50 with diff. The other side pane collapses to a single-line bar.
func (m Model) distributeNarrow(available int) (fileH, gitH, diffH int) {
	// Determine which side pane stays expanded.
	activeSide := m.focused
	if activeSide == PaneDiff {
		activeSide = m.prevFocused
	}

	remaining := available - collapsedHeight
	sideH := remaining / 2
	diffH = remaining - sideH

	if activeSide == PaneFileList {
		fileH = sideH
		gitH = collapsedHeight
	} else {
		gitH = sideH
		fileH = collapsedHeight
	}

	// Safety: ensure diff always has at least paneChrome height.
	if diffH < paneChrome {
		diffH = paneChrome
	}

	return fileH, gitH, diffH
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
			inactive := lipgloss.NewStyle().Bold(true)
			titleStr = inactive.Render(hotkey) + dash + inactive.Render(rest)
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
	// Show "No matches" when filter is active with zero results
	if m.focused == PaneFileList &&
		m.fileList.IsFiltering() &&
		m.fileList.FilterQuery() != "" &&
		!m.fileList.FilterHasMatches() {
		return StatusBarError.Width(m.width).Render("No matches")
	}

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
	// Filter typing mode replaces the footer with a text input
	if m.focused == PaneFileList && m.fileList.IsFiltering() {
		fi := m.fileList.filterInput
		fi.Width = max(0, m.width-4)
		return " " + fi.View()
	}

	// Locked filter mode shows a persistent indicator
	if m.focused == PaneFileList && m.fileList.IsFilterLocked() {
		query := m.fileList.FilterQuery()
		count := m.fileList.FileCount()
		indicator := HelpDesc.Render(fmt.Sprintf(" Filter: %d matches for ", count)) +
			HelpKey.Render("'"+query+"'") +
			HelpSep.Render(" | ") +
			HelpKey.Render("<esc>") + " " + HelpDesc.Render("exit filter mode")
		return indicator
	}

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
				hint("D", "discard"), hint("+", "new"), hint("e", "edit"),
				hint("/", "filter"),
			}
		case sel != nil && sel.Drift == DriftSourceEdited:
			paneHints = []string{
				hint("a", "apply (source → dest)"), hint("space", "add"),
				hint("D", "discard"), hint("+", "new"), hint("e", "edit"),
				hint("/", "filter"),
			}
		default:
			paneHints = []string{
				hint("space", "add"), hint("a", "apply"),
				hint("D", "discard"), hint("+", "new"), hint("e", "edit"),
				hint("/", "filter"),
			}
		}
	case PaneGitStatus:
		paneHints = []string{
			hint("space", "stage"), hint("a", "stage all"),
			hint("c", "commit"), hint("p", "pull"), hint("P", "push"), hint("D", "discard"),
		}
	case PaneDiff:
		paneHints = []string{hint("esc", "back")}
	}

	globalHints := []string{
		hint("C", "config"), hint("?", "help"), hint("q", "quit"),
	}

	allHints := append(paneHints, globalHints...)
	right := hyperlink("https://github.com/nickrotondo/lazychez", FooterLink.Render("lazychez")) + " " + lipgloss.NewStyle().Foreground(TextColor).Render(m.version) + " "
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

func (m Model) helpContent() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(TextColor)
	heading := lipgloss.NewStyle().Foreground(ActiveBorderColor)

	return title.Render("Keybindings") + "\n\n" +
		heading.Render("  Navigation") + `
    j/k         Move up/down
    g/G         Jump to top/bottom
    ctrl+d/u    Half-page down/up
    H/L         Previous/next pane
    0/1/2       Jump to panel
    ←/→         Cycle between file list and git
    tab         Next panel
    shift+tab   Previous panel
    esc         Back from diff panel
` + "\n" +
		heading.Render("  File Actions") + `
    space       Add file (dest → source)
    a           Apply file (source → dest)
    A           Apply all files
    D           Discard drift (revert change)
    e           Edit source (chezmoi edit)
    E           Edit destination file
    +           Add unmanaged file
    x           Forget file (unmanage)
` + "\n" +
		heading.Render("  Git Actions") + `
    space       Stage file (git add)
    a           Stage all files
    c           Commit (opens input)
    p           Pull from remote
    P           Push to remote
    D           Discard changes
` + "\n" +
		heading.Render("  Filter (File List)") + `
    /           Start filtering files
    enter       Lock filter (navigate matches)
    esc         Cancel / exit filter mode
` + "\n" +
		heading.Render("  General") + `
    r           Refresh all panes
    C           Edit chezmoi config
    ?           Toggle this help
    q           Quit`
}

func (m Model) renderHelp() string {
	content := m.helpViewport.View()
	total := m.helpViewport.TotalLineCount()
	visibleH := m.helpViewport.Height

	if total <= visibleH || visibleH <= 0 {
		return OverlayStyle.Render(content)
	}

	// Add scrollbar on the right edge
	thumbSize := max(1, visibleH*visibleH/total)
	maxOff := total - visibleH
	thumbStart := m.helpViewport.YOffset * (visibleH - thumbSize) / maxOff
	thumbEnd := thumbStart + thumbSize

	thumbStyle := lipgloss.NewStyle().Foreground(ActiveBorderColor)
	trackStyle := lipgloss.NewStyle().Foreground(MutedColor)

	lines := strings.Split(content, "\n")
	var b strings.Builder
	for i := 0; i < visibleH; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		if i >= thumbStart && i < thumbEnd {
			b.WriteString(line + " " + thumbStyle.Render("▐"))
		} else {
			b.WriteString(line + " " + trackStyle.Render("│"))
		}
		if i < visibleH-1 {
			b.WriteByte('\n')
		}
	}

	return OverlayStyle.Render(b.String())
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
	hint := HelpKey.Render("space") + " " + HelpDesc.Render("select") +
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
		return "0"
	}
	return fmt.Sprintf("%d of %d", current, total)
}

// hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence.
func hyperlink(url, text string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}
