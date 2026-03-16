package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nickrotondo/lazychez/internal/chezmoi"
	"github.com/nickrotondo/lazychez/internal/git"
)

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

func fetchDiff(r chezmoi.Runner, path string, reverse bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		diff, err := r.Diff(ctx, path)
		if err == nil && reverse {
			diff = reverseDiff(diff)
		}
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

func unstageFile(r git.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := r.Reset(ctx, path)
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

func pullFromRemote(r git.Runner) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := r.Pull(ctx)
		return PullResultMsg{Err: err}
	}
}

func restoreFile(r git.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := r.Restore(ctx, path)
		return GitDiscardResultMsg{Path: path, Err: err}
	}
}

func cleanFile(r git.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := r.Clean(ctx, path)
		return GitDiscardResultMsg{Path: path, Err: err}
	}
}

// reverseDiff swaps +/- in a unified diff so additions become removals and
// vice versa. Used to show dest-edited diffs from the user's perspective
// (what "add" would capture) instead of chezmoi's default (what "apply" would do).
func reverseDiff(diff string) string {
	var b strings.Builder
	b.Grow(len(diff))
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "--- "):
			b.WriteString("+++ " + line[4:])
		case strings.HasPrefix(line, "+++ "):
			b.WriteString("--- " + line[4:])
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			b.WriteString("+" + line[1:])
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			b.WriteString("-" + line[1:])
		default:
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	// Trim trailing extra newline from Split
	result := b.String()
	if strings.HasSuffix(diff, "\n") {
		result = strings.TrimRight(result, "\n") + "\n"
	} else {
		result = strings.TrimRight(result, "\n")
	}
	return result
}

func clearStatusAfter() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}
