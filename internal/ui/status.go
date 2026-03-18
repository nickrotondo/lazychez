package ui

import "fmt"

// StatusModel renders a static summary pane (branch, ahead/behind).
// It has no cursor or scrolling — it simply displays a single line of text.
type StatusModel struct {
	branch  string
	remote  string
	ahead   int
	behind  int
	loaded  bool
	focused bool
	width   int
	height  int
}

func NewStatusModel() StatusModel {
	return StatusModel{}
}

func (s *StatusModel) SetAheadBehind(ahead, behind int, branch, remote string) {
	s.ahead = ahead
	s.behind = behind
	s.branch = branch
	s.remote = remote
	s.loaded = true
}

func (s *StatusModel) SetDimensions(w, h int) {
	s.width = w
	s.height = h
}

func (s *StatusModel) SetFocused(focused bool) {
	s.focused = focused
}

func (s StatusModel) View() string {
	if !s.loaded {
		return "loading…"
	}
	return FormatAheadBehind(s.ahead, s.behind, s.branch, s.remote)
}

// FormatAheadBehind renders the status line: "↑N ↓M branch → remote".
// Zero counts hide the corresponding arrow. No upstream shows "branch (no remote)".
func FormatAheadBehind(ahead, behind int, branch, remote string) string {
	if remote == "" {
		return fmt.Sprintf("%s (no remote)", branch)
	}

	var parts []string
	if ahead > 0 {
		parts = append(parts, fmt.Sprintf("↑%d", ahead))
	}
	if behind > 0 {
		parts = append(parts, fmt.Sprintf("↓%d", behind))
	}
	parts = append(parts, fmt.Sprintf("%s → %s", branch, remote))

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
}

func (s StatusModel) ScrollState() (int, int) {
	return 0, 0
}
