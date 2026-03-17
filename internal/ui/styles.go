package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Border colors
	ActiveBorderColor   = lipgloss.Color("#81cae4")
	InactiveBorderColor = lipgloss.Color("#5a7a86")

	// Semantic colors
	ModifiedColor = lipgloss.Color("#e4cd81")
	AddedColor    = lipgloss.Color("#98e481")
	DeletedColor  = lipgloss.Color("#e48281")
	TitleColor    = lipgloss.Color("#81cae4")
	SelectedBg    = lipgloss.Color("#114a5f")
	MutedColor    = lipgloss.Color("#5a7a86")
	SuccessColor  = lipgloss.Color("#98e481")
	ErrorColor    = lipgloss.Color("#e48281")

	// Diff colors
	DiffAddColor  = lipgloss.Color("#98e481")
	DiffDelColor  = lipgloss.Color("#e48281")
	DiffHunkColor = lipgloss.Color("#9c81e4")
	DiffMetaColor = lipgloss.Color("#5a7a86")

	// Pane styles
	ActivePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ActiveBorderColor)

	InactivePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(InactiveBorderColor)

	// Title style (rendered as first line inside pane)
	PaneTitle = lipgloss.NewStyle().
			Foreground(TitleColor).
			Bold(true)

	// File list
	SelectedItem = lipgloss.NewStyle().
			Background(SelectedBg).
			Bold(true)

	NormalItem = lipgloss.NewStyle()

	// Status indicators
	SourceEditedIndicator = lipgloss.NewStyle().Foreground(ModifiedColor).SetString("●") // source changed → apply
	DestEditedIndicator   = lipgloss.NewStyle().Foreground(TitleColor).SetString("◆")   // dest changed → add
	AddedIndicator       = lipgloss.NewStyle().Foreground(AddedColor).SetString("+")
	DeletedIndicator     = lipgloss.NewStyle().Foreground(DeletedColor).SetString("−")

	// Diff line styles
	DiffAdd  = lipgloss.NewStyle().Foreground(DiffAddColor)
	DiffDel  = lipgloss.NewStyle().Foreground(DiffDelColor)
	DiffHunk = lipgloss.NewStyle().Foreground(DiffHunkColor)
	DiffMeta = lipgloss.NewStyle().Foreground(DiffMetaColor)

	// Footer
	HelpKey  = lipgloss.NewStyle().Foreground(ActiveBorderColor).Bold(true)
	HelpDesc = lipgloss.NewStyle().Foreground(MutedColor)
	HelpSep  = lipgloss.NewStyle().Foreground(MutedColor)
	FooterLink = lipgloss.NewStyle().Foreground(lipgloss.Color("#e9f6fb"))

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e9f6fb"))
	StatusBarError = lipgloss.NewStyle().
			Foreground(ErrorColor)
	StatusBarSuccess = lipgloss.NewStyle().
				Foreground(SuccessColor)

	// Overlay (help, commit input, confirm dialogs)
	OverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ActiveBorderColor).
			Padding(1, 2).
			Background(lipgloss.Color("#172b32"))
)
