package ui

import "github.com/nickrotondo/lazychez/internal/chezmoi"

// Data fetch results

type ManagedFilesMsg struct {
	Files []chezmoi.ManagedFile
	Err   error
}

type StatusMsg struct {
	Entries []chezmoi.StatusEntry
	Err     error
}

type DiffMsg struct {
	Path string
	Diff string
	Err  error
}

// Action results

type AddResultMsg struct {
	Path string
	Err  error
}

type ApplyResultMsg struct {
	Path string
	Err  error
}

type ApplyAllResultMsg struct {
	Err error
}

type ForgetResultMsg struct {
	Path string
	Err  error
}

type UnmanagedMsg struct {
	Files []string
	Err   error
}

type AddNewFileResultMsg struct {
	Path string
	Err  error
}

type BatchAddResultMsg struct {
	Added  []string // paths that were added successfully
	Errors map[string]error
}

type GitStatusMsg struct {
	Entries []GitStatusEntry
	Err     error
}

type GitStageResultMsg struct {
	Path string // empty when staging all
	Err  error
}

type CommitResultMsg struct {
	Err error
}

type PushResultMsg struct {
	Err error
}

type PullResultMsg struct {
	Err error
}

type GitDiscardResultMsg struct {
	Path string
	Err  error
}

type EditorFinishedMsg struct {
	Err error
}

type AheadBehindMsg struct {
	Ahead  int
	Behind int
	Branch string
	Remote string
	Err    error
}

// UI messages

type ClearStatusMsg struct{}

type TickMsg struct{}
