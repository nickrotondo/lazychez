package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Border colors
	ActiveBorderColor   = lipgloss.Color("#7aa2f7")
	InactiveBorderColor = lipgloss.Color("#565f89")

	// Semantic colors
	ModifiedColor = lipgloss.Color("#e0af68")
	AddedColor    = lipgloss.Color("#9ece6a")
	DeletedColor  = lipgloss.Color("#f7768e")
	TitleColor    = lipgloss.Color("#7aa2f7")
	SelectedBg    = lipgloss.Color("#283457")
	MutedColor    = lipgloss.Color("#565f89")
	SuccessColor  = lipgloss.Color("#9ece6a")
	ErrorColor    = lipgloss.Color("#f7768e")

	// Diff colors
	DiffAddColor  = lipgloss.Color("#9ece6a")
	DiffDelColor  = lipgloss.Color("#f7768e")
	DiffHunkColor = lipgloss.Color("#bb9af7")
	DiffMetaColor = lipgloss.Color("#565f89")

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
	ModifiedIndicator    = lipgloss.NewStyle().Foreground(ModifiedColor).SetString("●")
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

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a9b1d6"))
	StatusBarError = lipgloss.NewStyle().
			Foreground(ErrorColor)
	StatusBarSuccess = lipgloss.NewStyle().
				Foreground(SuccessColor)

	// Overlay (help, commit input, confirm dialogs)
	OverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ActiveBorderColor).
			Padding(1, 2).
			Background(lipgloss.Color("#1a1b26"))
)
