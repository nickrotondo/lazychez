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

// UI messages

type StatusBarMsg struct {
	Text    string
	IsError bool
}

type ClearStatusMsg struct{}
