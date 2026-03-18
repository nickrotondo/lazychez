package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nickrotondo/lazychez/internal/chezmoi"
)

// --- Message handling tests ---

func TestUpdate_ManagedFilesMsg(t *testing.T) {
	t.Run("success stores files and rebuilds list", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := ManagedFilesMsg{
			Files: []chezmoi.ManagedFile{
				{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
				{Path: ".vimrc", SourceRelPath: "dot_vimrc"},
			},
		}
		result, _ := m.Update(msg)
		m = result.(Model)
		if m.fileList.FileCount() != 2 {
			t.Errorf("FileCount() = %d, want 2", m.fileList.FileCount())
		}
	})

	t.Run("error sets status", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := ManagedFilesMsg{Err: fmt.Errorf("chezmoi failed")}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
		if !strings.Contains(m.statusMsg, "Error") {
			t.Errorf("statusMsg = %q, want to contain 'Error'", m.statusMsg)
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatusAfter)")
		}
	})
}

func TestUpdate_StatusMsg(t *testing.T) {
	t.Run("success merges with managed files", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.managedFiles = []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		msg := StatusMsg{
			Entries: []chezmoi.StatusEntry{
				{SourceState: ' ', DestState: 'M', Path: ".zshrc"},
			},
		}
		result, _ := m.Update(msg)
		m = result.(Model)
		// Should have rebuilt with drift
		if m.fileList.DriftCount() != 1 {
			t.Errorf("DriftCount() = %d, want 1", m.fileList.DriftCount())
		}
	})

	t.Run("error sets status", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := StatusMsg{Err: fmt.Errorf("status failed")}
		result, _ := m.Update(msg)
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
	})
}

func TestUpdate_DiffMsg(t *testing.T) {
	t.Run("success populates cache", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := DiffMsg{Path: ".zshrc", Diff: "+new line\n-old line\n"}
		result, _ := m.Update(msg)
		m = result.(Model)
		if m.diffCache[".zshrc"] != msg.Diff {
			t.Errorf("diffCache[.zshrc] = %q, want %q", m.diffCache[".zshrc"], msg.Diff)
		}
	})

	t.Run("error shows in diff view", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := DiffMsg{Path: ".zshrc", Err: fmt.Errorf("diff failed")}
		result, _ := m.Update(msg)
		m = result.(Model)
		// diffView should have content set (the error message)
		if m.diffView.path != ".zshrc" {
			t.Errorf("diffView.path = %q, want .zshrc", m.diffView.path)
		}
	})
}

func TestUpdate_GitStatusMsg(t *testing.T) {
	t.Run("success sets entries", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := GitStatusMsg{
			Entries: []GitStatusEntry{
				{XY: " M", Path: "dot_zshrc"},
				{XY: "??", Path: "new_file"},
			},
		}
		result, _ := m.Update(msg)
		m = result.(Model)
		if m.gitStatus.EntryCount() != 2 {
			t.Errorf("EntryCount() = %d, want 2", m.gitStatus.EntryCount())
		}
	})

	t.Run("error sets status", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := GitStatusMsg{Err: fmt.Errorf("git failed")}
		result, _ := m.Update(msg)
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
	})
}

func TestUpdate_AddResultMsg(t *testing.T) {
	t.Run("success sets status", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := AddResultMsg{Path: ".zshrc"}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Added") {
			t.Errorf("statusMsg = %q, want to contain 'Added'", m.statusMsg)
		}
		if m.statusError {
			t.Error("statusError should be false on success")
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatus + refresh)")
		}
	})

	t.Run("error sets status", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := AddResultMsg{Path: ".zshrc", Err: fmt.Errorf("add failed")}
		result, _ := m.Update(msg)
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
		if !strings.Contains(m.statusMsg, "Error") {
			t.Errorf("statusMsg = %q, want to contain 'Error'", m.statusMsg)
		}
	})

	t.Run("template error shows special message", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := AddResultMsg{
			Path: ".zshrc",
			Err:  &chezmoi.TemplateEditError{Path: ".zshrc"},
		}
		result, _ := m.Update(msg)
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "template") {
			t.Errorf("statusMsg = %q, want to contain 'template'", m.statusMsg)
		}
	})
}

func TestUpdate_ApplyResultMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(ApplyResultMsg{Path: ".zshrc"})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Applied") {
			t.Errorf("statusMsg = %q, want to contain 'Applied'", m.statusMsg)
		}
	})

	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(ApplyResultMsg{Path: ".zshrc", Err: fmt.Errorf("fail")})
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
	})
}

func TestUpdate_ApplyAllResultMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(ApplyAllResultMsg{})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Applied all") {
			t.Errorf("statusMsg = %q, want to contain 'Applied all'", m.statusMsg)
		}
	})

	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(ApplyAllResultMsg{Err: fmt.Errorf("fail")})
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
	})
}

func TestUpdate_CommitResultMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(CommitResultMsg{})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Committed") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
	})

	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(CommitResultMsg{Err: fmt.Errorf("fail")})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Commit failed") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
	})
}

func TestUpdate_PushResultMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(PushResultMsg{})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Pushed") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
	})

	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(PushResultMsg{Err: fmt.Errorf("fail")})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Push failed") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
	})
}

func TestUpdate_PullResultMsg(t *testing.T) {
	t.Run("success triggers refresh", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, cmd := m.Update(PullResultMsg{})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Pulled") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatus + refresh)")
		}
	})

	t.Run("error does not refresh", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, cmd := m.Update(PullResultMsg{Err: fmt.Errorf("fail")})
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
		// cmd is clearStatusAfter only, not a batch with refresh
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatusAfter)")
		}
	})
}

func TestUpdate_GitDiscardResultMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, cmd := m.Update(GitDiscardResultMsg{Path: "file.txt"})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Discarded") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
		if cmd == nil {
			t.Error("cmd should not be nil")
		}
	})

	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(GitDiscardResultMsg{Path: "file.txt", Err: fmt.Errorf("fail")})
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
	})
}

func TestUpdate_ForgetResultMsg(t *testing.T) {
	t.Run("success shows status and refreshes", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, cmd := m.Update(ForgetResultMsg{Path: ".zshrc"})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Forgot .zshrc") {
			t.Errorf("statusMsg = %q, want to contain 'Forgot .zshrc'", m.statusMsg)
		}
		if m.statusError {
			t.Error("statusError should be false")
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatus + refresh)")
		}
	})

	t.Run("error shows error status", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, cmd := m.Update(ForgetResultMsg{Path: ".zshrc", Err: fmt.Errorf("forget failed")})
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
		if !strings.Contains(m.statusMsg, "Error forgetting .zshrc") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatus + refresh)")
		}
	})
}

func TestUpdate_GitStageResultMsg(t *testing.T) {
	t.Run("single file success", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(GitStageResultMsg{Path: "file.txt"})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Staged file.txt") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
	})

	t.Run("stage all success", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(GitStageResultMsg{})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Staged all") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
	})

	t.Run("single file error", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(GitStageResultMsg{Path: "file.txt", Err: fmt.Errorf("fail")})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Error staging file.txt") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
	})

	t.Run("stage all error", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(GitStageResultMsg{Err: fmt.Errorf("fail")})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Error staging all") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
	})
}

func TestUpdate_EditorFinishedMsg(t *testing.T) {
	t.Run("error sets status", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(EditorFinishedMsg{Err: fmt.Errorf("editor crashed")})
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
		if !strings.Contains(m.statusMsg, "Editor error") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
	})

	t.Run("success triggers refresh", func(t *testing.T) {
		m, _, _ := newTestModel()
		_, cmd := m.Update(EditorFinishedMsg{})
		if cmd == nil {
			t.Error("cmd should not be nil (refreshAll)")
		}
	})
}

func TestUpdate_ClearStatusMsg(t *testing.T) {
	m, _, _ := newTestModel()
	m.statusMsg = "some message"
	m.statusError = true
	result, _ := m.Update(ClearStatusMsg{})
	m = result.(Model)
	if m.statusMsg != "" {
		t.Errorf("statusMsg = %q, want empty", m.statusMsg)
	}
	if m.statusError {
		t.Error("statusError should be false")
	}
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m, _, _ := newTestModel()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	m = result.(Model)
	if m.width != 200 {
		t.Errorf("width = %d, want 200", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
}

// --- Key handling tests ---

func sendKey(m Model, key string) (Model, tea.Cmd) {
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return result.(Model), cmd
}

func sendSpecialKey(m Model, keyType tea.KeyType) (Model, tea.Cmd) {
	result, cmd := m.Update(tea.KeyMsg{Type: keyType})
	return result.(Model), cmd
}

func TestHandleKey_GlobalKeys(t *testing.T) {
	t.Run("q quits", func(t *testing.T) {
		m, _, _ := newTestModel()
		_, cmd := sendKey(m, "q")
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("cmd() returned %T, want tea.QuitMsg", msg)
		}
	})

	t.Run("tab cycles focus forward", func(t *testing.T) {
		m, _, _ := newTestModel()
		if m.focused != PaneFileList {
			t.Fatalf("initial focus = %d, want PaneFileList", m.focused)
		}
		m, _ = sendSpecialKey(m, tea.KeyTab)
		if m.focused != PaneGitStatus {
			t.Errorf("after tab: focused = %d, want PaneGitStatus", m.focused)
		}
		m, _ = sendSpecialKey(m, tea.KeyTab)
		if m.focused != PaneDiff {
			t.Errorf("after tab: focused = %d, want PaneDiff", m.focused)
		}
		m, _ = sendSpecialKey(m, tea.KeyTab)
		if m.focused != PaneFileList {
			t.Errorf("after tab: focused = %d, want PaneFileList (wrap)", m.focused)
		}
	})

	t.Run("shift+tab cycles focus backward", func(t *testing.T) {
		m, _, _ := newTestModel()
		m, _ = sendSpecialKey(m, tea.KeyShiftTab)
		if m.focused != PaneDiff {
			t.Errorf("after shift+tab: focused = %d, want PaneDiff", m.focused)
		}
	})

	t.Run("1 focuses file list", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneDiff
		m, _ = sendKey(m, "1")
		if m.focused != PaneFileList {
			t.Errorf("focused = %d, want PaneFileList", m.focused)
		}
	})

	t.Run("2 focuses git status", func(t *testing.T) {
		m, _, _ := newTestModel()
		m, _ = sendKey(m, "2")
		if m.focused != PaneGitStatus {
			t.Errorf("focused = %d, want PaneGitStatus", m.focused)
		}
	})

	t.Run("0 focuses diff", func(t *testing.T) {
		m, _, _ := newTestModel()
		m, _ = sendKey(m, "0")
		if m.focused != PaneDiff {
			t.Errorf("focused = %d, want PaneDiff", m.focused)
		}
	})

	t.Run("left/right cycles between file list and git status", func(t *testing.T) {
		m, _, _ := newTestModel()
		// Start at file list, right arrow → git status
		m, _ = sendSpecialKey(m, tea.KeyRight)
		if m.focused != PaneGitStatus {
			t.Errorf("after right: focused = %d, want PaneGitStatus", m.focused)
		}
		// Right again → back to file list
		m, _ = sendSpecialKey(m, tea.KeyRight)
		if m.focused != PaneFileList {
			t.Errorf("after right: focused = %d, want PaneFileList", m.focused)
		}
		// Left from file list → git status
		m, _ = sendSpecialKey(m, tea.KeyLeft)
		if m.focused != PaneGitStatus {
			t.Errorf("after left: focused = %d, want PaneGitStatus", m.focused)
		}
		// From diff pane, arrow goes to file list
		m.focused = PaneDiff
		m, _ = sendSpecialKey(m, tea.KeyLeft)
		if m.focused != PaneFileList {
			t.Errorf("after left from diff: focused = %d, want PaneFileList", m.focused)
		}
	})

	t.Run("r triggers refresh", func(t *testing.T) {
		m, _, _ := newTestModel()
		_, cmd := sendKey(m, "r")
		if cmd == nil {
			t.Error("cmd should not be nil (refreshAll)")
		}
	})

	t.Run("question mark opens help overlay", func(t *testing.T) {
		m, _, _ := newTestModel()
		m, _ = sendKey(m, "?")
		if m.overlay != OverlayHelp {
			t.Errorf("overlay = %d, want OverlayHelp", m.overlay)
		}
	})
}

func TestHandleKey_OverlayHelp(t *testing.T) {
	setup := func() Model {
		m, _, _ := newTestModel()
		m.overlay = OverlayHelp
		return m
	}

	t.Run("question mark closes help", func(t *testing.T) {
		m := setup()
		m, _ = sendKey(m, "?")
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
	})

	t.Run("esc closes help", func(t *testing.T) {
		m := setup()
		m, _ = sendSpecialKey(m, tea.KeyEscape)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
	})

	t.Run("q closes help", func(t *testing.T) {
		m := setup()
		m, _ = sendKey(m, "q")
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
	})

	t.Run("other keys are no-op", func(t *testing.T) {
		m := setup()
		m, _ = sendKey(m, "j")
		if m.overlay != OverlayHelp {
			t.Errorf("overlay = %d, want OverlayHelp (unchanged)", m.overlay)
		}
	})
}

func TestHandleKey_OverlayCommit(t *testing.T) {
	setup := func() Model {
		m, _, _ := newTestModel()
		m.overlay = OverlayCommit
		m.commitInput.Focus()
		return m
	}

	t.Run("esc cancels", func(t *testing.T) {
		m := setup()
		m, cmd := sendSpecialKey(m, tea.KeyEscape)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if cmd != nil {
			t.Error("cmd should be nil (no commit)")
		}
	})

	t.Run("enter with empty message is no-op", func(t *testing.T) {
		m := setup()
		m, cmd := sendSpecialKey(m, tea.KeyEnter)
		if m.overlay != OverlayCommit {
			t.Errorf("overlay = %d, want OverlayCommit (unchanged)", m.overlay)
		}
		if cmd != nil {
			t.Error("cmd should be nil")
		}
	})

	t.Run("enter with message commits", func(t *testing.T) {
		m := setup()
		m.commitInput.SetValue("test commit")
		m, cmd := sendSpecialKey(m, tea.KeyEnter)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if cmd == nil {
			t.Fatal("cmd should not be nil (commitChanges)")
		}
		// Execute the command — it should call the mock git runner
		msg := cmd()
		if _, ok := msg.(CommitResultMsg); !ok {
			t.Errorf("cmd() returned %T, want CommitResultMsg", msg)
		}
	})
}

func TestHandleKey_OverlayConfirmApplyAll(t *testing.T) {
	setup := func() Model {
		m, _, _ := newTestModel()
		m.overlay = OverlayConfirmApplyAll
		return m
	}

	t.Run("y applies", func(t *testing.T) {
		m := setup()
		m, cmd := sendKey(m, "y")
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if _, ok := msg.(ApplyAllResultMsg); !ok {
			t.Errorf("cmd() returned %T, want ApplyAllResultMsg", msg)
		}
	})

	t.Run("n cancels", func(t *testing.T) {
		m := setup()
		m, _ = sendKey(m, "n")
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
	})

	t.Run("esc cancels", func(t *testing.T) {
		m := setup()
		m, _ = sendSpecialKey(m, tea.KeyEscape)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
	})
}

func TestHandleKey_OverlayConfirmGitDiscard(t *testing.T) {
	t.Run("y with tracked file restores", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.overlay = OverlayConfirmGitDiscard
		m.discardPath = "file.txt"
		m.discardUntracked = false
		m, cmd := sendKey(m, "y")
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if result, ok := msg.(GitDiscardResultMsg); !ok {
			t.Errorf("cmd() returned %T, want GitDiscardResultMsg", msg)
		} else if result.Path != "file.txt" {
			t.Errorf("result.Path = %q, want file.txt", result.Path)
		}
	})

	t.Run("y with untracked file cleans", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.overlay = OverlayConfirmGitDiscard
		m.discardPath = "new_file.txt"
		m.discardUntracked = true
		m, cmd := sendKey(m, "y")
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if result, ok := msg.(GitDiscardResultMsg); !ok {
			t.Errorf("cmd() returned %T, want GitDiscardResultMsg", msg)
		} else if result.Path != "new_file.txt" {
			t.Errorf("result.Path = %q, want new_file.txt", result.Path)
		}
	})

	t.Run("n cancels", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.overlay = OverlayConfirmGitDiscard
		m, _ = sendKey(m, "n")
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
	})
}

func TestHandleKey_OverlayConfirmForget(t *testing.T) {
	setup := func() (Model, *mockChezmoiRunner) {
		m, cm, _ := newTestModel()
		m.overlay = OverlayConfirmForget
		m.forgetPath = ".zshrc"
		return m, cm
	}

	t.Run("y confirms forget", func(t *testing.T) {
		m, _ := setup()
		m, cmd := sendKey(m, "y")
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		result, ok := msg.(ForgetResultMsg)
		if !ok {
			t.Fatalf("cmd() returned %T, want ForgetResultMsg", msg)
		}
		if result.Path != ".zshrc" {
			t.Errorf("result.Path = %q, want .zshrc", result.Path)
		}
	})

	t.Run("n cancels", func(t *testing.T) {
		m, _ := setup()
		m, cmd := sendKey(m, "n")
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if cmd != nil {
			t.Error("cmd should be nil (no forget)")
		}
	})

	t.Run("esc cancels", func(t *testing.T) {
		m, _ := setup()
		m, cmd := sendSpecialKey(m, tea.KeyEscape)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if cmd != nil {
			t.Error("cmd should be nil (no forget)")
		}
	})
}

func TestUpdate_UnmanagedMsg(t *testing.T) {
	t.Run("success opens add file overlay", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := UnmanagedMsg{Files: []string{".config/new", ".profile"}}
		result, _ := m.Update(msg)
		m = result.(Model)
		if m.overlay != OverlayAddFile {
			t.Errorf("overlay = %d, want OverlayAddFile", m.overlay)
		}
	})

	t.Run("empty list shows status, no overlay", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := UnmanagedMsg{Files: []string{}}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if m.statusMsg != "No unmanaged files" {
			t.Errorf("statusMsg = %q, want 'No unmanaged files'", m.statusMsg)
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatusAfter)")
		}
	})

	t.Run("nil files shows status, no overlay", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := UnmanagedMsg{Files: nil}
		result, _ := m.Update(msg)
		m = result.(Model)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
	})

	t.Run("error sets error status", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := UnmanagedMsg{Err: fmt.Errorf("unmanaged failed")}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
		if !strings.Contains(m.statusMsg, "Error") {
			t.Errorf("statusMsg = %q, want to contain 'Error'", m.statusMsg)
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatusAfter)")
		}
	})
}

func TestUpdate_AddNewFileResultMsg(t *testing.T) {
	t.Run("success shows status and refreshes", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := AddNewFileResultMsg{Path: ".profile"}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Added .profile") {
			t.Errorf("statusMsg = %q, want to contain 'Added .profile'", m.statusMsg)
		}
		if m.statusError {
			t.Error("statusError should be false")
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatus + refresh)")
		}
	})

	t.Run("error shows error status", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := AddNewFileResultMsg{Path: ".profile", Err: fmt.Errorf("add failed")}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
		if !strings.Contains(m.statusMsg, "Error adding .profile") {
			t.Errorf("statusMsg = %q", m.statusMsg)
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatus + refresh)")
		}
	})
}

func TestHandleKey_OverlayAddFile(t *testing.T) {
	setup := func() Model {
		m, _, _ := newTestModel()
		m.addFile = NewAddFileModel([]string{".config/new", ".profile", ".local/share/app"}, 50, 15)
		m.overlay = OverlayAddFile
		return m
	}

	t.Run("esc closes overlay when not filtering", func(t *testing.T) {
		m := setup()
		m, cmd := sendSpecialKey(m, tea.KeyEscape)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if cmd != nil {
			t.Error("cmd should be nil")
		}
	})

	t.Run("enter adds selected file", func(t *testing.T) {
		m := setup()
		m, cmd := sendSpecialKey(m, tea.KeyEnter)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		result, ok := msg.(AddNewFileResultMsg)
		if !ok {
			t.Fatalf("cmd() returned %T, want AddNewFileResultMsg", msg)
		}
		if result.Path != ".config/new" {
			t.Errorf("result.Path = %q, want .config/new", result.Path)
		}
	})

	t.Run("enter with no items is no-op", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.addFile = NewAddFileModel([]string{}, 50, 15)
		m.overlay = OverlayAddFile
		m, cmd := sendSpecialKey(m, tea.KeyEnter)
		if m.overlay != OverlayAddFile {
			t.Errorf("overlay = %d, want OverlayAddFile (unchanged)", m.overlay)
		}
		if cmd != nil {
			t.Error("cmd should be nil (no item selected)")
		}
	})

	t.Run("arrow keys navigate the list", func(t *testing.T) {
		m := setup()
		first := m.addFile.SelectedPath()
		m, _ = sendSpecialKey(m, tea.KeyDown)
		second := m.addFile.SelectedPath()
		if first == second {
			t.Error("cursor should have moved down")
		}
		m, _ = sendSpecialKey(m, tea.KeyUp)
		third := m.addFile.SelectedPath()
		if third != first {
			t.Errorf("after up: path = %q, want %q (back to first)", third, first)
		}
	})

	t.Run("typing filters the list", func(t *testing.T) {
		m := setup()
		// Type "profile" to filter — filter is auto-focused
		for _, r := range "profile" {
			m, _ = sendKey(m, string(r))
		}
		path := m.addFile.SelectedPath()
		if path != ".profile" {
			t.Errorf("after filtering: selected = %q, want .profile", path)
		}
	})

	t.Run("space toggles selection on current item", func(t *testing.T) {
		m := setup()
		if m.addFile.SelectionCount() != 0 {
			t.Fatalf("initial selection count = %d, want 0", m.addFile.SelectionCount())
		}
		m, _ = sendSpecialKey(m, tea.KeySpace)
		if m.addFile.SelectionCount() != 1 {
			t.Errorf("selection count after space = %d, want 1", m.addFile.SelectionCount())
		}
		if !m.addFile.selected[".config/new"] {
			t.Error("expected .config/new to be selected")
		}
		// Cursor stays on the same item
		if m.addFile.SelectedPath() != ".config/new" {
			t.Errorf("cursor should stay on .config/new, got %q", m.addFile.SelectedPath())
		}
		// Toggle again to deselect
		m, _ = sendSpecialKey(m, tea.KeySpace)
		if m.addFile.SelectionCount() != 0 {
			t.Errorf("selection count after deselect = %d, want 0", m.addFile.SelectionCount())
		}
	})

	t.Run("enter with multi-select closes overlay and dispatches batch add", func(t *testing.T) {
		m := setup()
		// Select first two files
		m, _ = sendSpecialKey(m, tea.KeySpace)
		m, _ = sendSpecialKey(m, tea.KeyDown)
		m, _ = sendSpecialKey(m, tea.KeySpace)
		if m.addFile.SelectionCount() != 2 {
			t.Fatalf("selection count = %d, want 2", m.addFile.SelectionCount())
		}
		m, cmd := sendSpecialKey(m, tea.KeyEnter)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		result, ok := msg.(BatchAddResultMsg)
		if !ok {
			t.Fatalf("cmd() returned %T, want BatchAddResultMsg", msg)
		}
		if len(result.Added) != 2 {
			t.Errorf("added count = %d, want 2", len(result.Added))
		}
	})

	t.Run("enter with no selections uses focused file fallback", func(t *testing.T) {
		m := setup()
		// No space toggles — just press enter
		m, cmd := sendSpecialKey(m, tea.KeyEnter)
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone (single-select fallback closes)", m.overlay)
		}
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		result, ok := msg.(AddNewFileResultMsg)
		if !ok {
			t.Fatalf("cmd() returned %T, want AddNewFileResultMsg", msg)
		}
		if result.Path != ".config/new" {
			t.Errorf("result.Path = %q, want .config/new", result.Path)
		}
	})
}

func TestUpdate_BatchAddResultMsg(t *testing.T) {
	setup := func() Model {
		m, _, _ := newTestModel()
		m.addFile = NewAddFileModel([]string{".config/new", ".profile", ".local/share/app"}, 50, 15)
		m.overlay = OverlayAddFile
		return m
	}

	t.Run("success shows status and refreshes", func(t *testing.T) {
		m := setup()
		msg := BatchAddResultMsg{
			Added:  []string{".config/new", ".profile"},
			Errors: map[string]error{},
		}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Added 2 files") {
			t.Errorf("statusMsg = %q, want to contain 'Added 2 files'", m.statusMsg)
		}
		if m.statusError {
			t.Error("statusError should be false")
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatus + refresh)")
		}
	})

	t.Run("partial failure shows error status", func(t *testing.T) {
		m := setup()
		msg := BatchAddResultMsg{
			Added:  []string{".config/new"},
			Errors: map[string]error{".profile": fmt.Errorf("permission denied")},
		}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
		if !strings.Contains(m.statusMsg, "1 failed") {
			t.Errorf("statusMsg = %q, want to contain '1 failed'", m.statusMsg)
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatus + refresh)")
		}
	})

	t.Run("refreshes all panes after batch add", func(t *testing.T) {
		m := setup()
		msg := BatchAddResultMsg{
			Added:  []string{".config/new"},
			Errors: map[string]error{},
		}
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Error("cmd should not be nil (should refresh)")
		}
	})
}

// --- Pane key handling ---

func TestHandleFileListKey(t *testing.T) {
	setupWithFiles := func() (Model, *mockChezmoiRunner) {
		m, cm, _ := newTestModel()
		// Populate with files that have drift
		m.managedFiles = []chezmoi.ManagedFile{
			{Path: ".bashrc", SourceRelPath: "dot_bashrc"},
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
			{Path: ".vimrc", SourceRelPath: "dot_vimrc"},
		}
		m.statusData = []chezmoi.StatusEntry{
			{SourceState: ' ', DestState: 'M', Path: ".bashrc"},
			{SourceState: 'M', DestState: ' ', Path: ".zshrc"},
		}
		m.gitStatus.entries = []GitStatusEntry{
			{XY: " M", Path: "dot_zshrc"},
		}
		m.rebuildFileList()
		m.updateDimensions()
		return m, cm
	}

	t.Run("j moves down", func(t *testing.T) {
		m, _ := setupWithFiles()
		startPath := m.fileList.SelectedPath()
		m, _ = sendKey(m, "j")
		newPath := m.fileList.SelectedPath()
		if newPath == startPath && m.fileList.FileCount() > 1 {
			t.Error("cursor did not move")
		}
	})

	t.Run("space triggers add on selected file", func(t *testing.T) {
		m, _ := setupWithFiles()
		path := m.fileList.SelectedPath()
		if path == "" {
			t.Fatal("no file selected")
		}
		_, cmd := sendKey(m, " ")
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if result, ok := msg.(AddResultMsg); !ok {
			t.Errorf("cmd() returned %T, want AddResultMsg", msg)
		} else if result.Path != path {
			t.Errorf("result.Path = %q, want %q", result.Path, path)
		}
	})

	t.Run("a triggers apply on selected file", func(t *testing.T) {
		m, _ := setupWithFiles()
		path := m.fileList.SelectedPath()
		_, cmd := sendKey(m, "a")
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if result, ok := msg.(ApplyResultMsg); !ok {
			t.Errorf("cmd() returned %T, want ApplyResultMsg", msg)
		} else if result.Path != path {
			t.Errorf("result.Path = %q, want %q", result.Path, path)
		}
	})

	t.Run("A opens confirm apply all overlay", func(t *testing.T) {
		m, _ := setupWithFiles()
		m, _ = sendKey(m, "A")
		if m.overlay != OverlayConfirmApplyAll {
			t.Errorf("overlay = %d, want OverlayConfirmApplyAll", m.overlay)
		}
	})

	t.Run("D on dest edited triggers apply", func(t *testing.T) {
		m, _ := setupWithFiles()
		// Navigate to a DriftDestEdited file
		for m.fileList.SelectedItem() == nil || m.fileList.SelectedItem().Drift != DriftDestEdited {
			m.fileList.MoveDown()
			if m.fileList.SelectedPath() == "" {
				t.Skip("could not find DriftDestEdited file")
			}
		}
		_, cmd := sendKey(m, "D")
		if cmd == nil {
			t.Fatal("cmd should not be nil for D on dest-edited")
		}
		msg := cmd()
		if _, ok := msg.(ApplyResultMsg); !ok {
			t.Errorf("cmd() returned %T, want ApplyResultMsg", msg)
		}
	})

	t.Run("D on source edited triggers add", func(t *testing.T) {
		m, _ := setupWithFiles()
		// Navigate to a DriftSourceEdited file
		for m.fileList.SelectedItem() == nil || m.fileList.SelectedItem().Drift != DriftSourceEdited {
			m.fileList.MoveDown()
			if m.fileList.SelectedPath() == "" {
				t.Skip("could not find DriftSourceEdited file")
			}
		}
		_, cmd := sendKey(m, "D")
		if cmd == nil {
			t.Fatal("cmd should not be nil for D on source-edited")
		}
		msg := cmd()
		if _, ok := msg.(AddResultMsg); !ok {
			t.Errorf("cmd() returned %T, want AddResultMsg", msg)
		}
	})

	t.Run("x opens confirm forget overlay", func(t *testing.T) {
		m, _ := setupWithFiles()
		path := m.fileList.SelectedPath()
		m, cmd := sendKey(m, "x")
		if m.overlay != OverlayConfirmForget {
			t.Errorf("overlay = %d, want OverlayConfirmForget", m.overlay)
		}
		if m.forgetPath != path {
			t.Errorf("forgetPath = %q, want %q", m.forgetPath, path)
		}
		if cmd != nil {
			t.Error("cmd should be nil (overlay opened, no command yet)")
		}
	})

	t.Run("+ triggers fetchUnmanaged", func(t *testing.T) {
		m, _ := setupWithFiles()
		m, cmd := sendKey(m, "+")
		if cmd == nil {
			t.Fatal("cmd should not be nil (fetchUnmanaged)")
		}
		if m.statusMsg != "Loading unmanaged files..." {
			t.Errorf("statusMsg = %q, want loading message", m.statusMsg)
		}
		msg := cmd()
		if _, ok := msg.(UnmanagedMsg); !ok {
			t.Errorf("cmd() returned %T, want UnmanagedMsg", msg)
		}
	})

	t.Run("x with no files is no-op", func(t *testing.T) {
		m, _, _ := newTestModel()
		m, _ = sendKey(m, "x")
		if m.overlay != OverlayNone {
			t.Errorf("overlay = %d, want OverlayNone", m.overlay)
		}
	})

	t.Run("space with no files is no-op", func(t *testing.T) {
		m, _, _ := newTestModel()
		_, cmd := sendKey(m, " ")
		if cmd != nil {
			t.Error("cmd should be nil when no file selected")
		}
	})
}

func TestHandleGitStatusKey(t *testing.T) {
	setupWithEntries := func() Model {
		m, _, _ := newTestModel()
		m.focused = PaneGitStatus
		m.syncFocus()
		m.gitStatus.SetEntries([]GitStatusEntry{
			{XY: " M", Path: "dot_zshrc"},
			{XY: "M ", Path: "dot_bashrc"},
			{XY: "??", Path: "new_file"},
		})
		m.gitStatus.SetDimensions(80, 20)
		return m
	}

	t.Run("space on unstaged file stages", func(t *testing.T) {
		m := setupWithEntries()
		// cursor at 0: XY=" M" (unstaged)
		_, cmd := sendKey(m, " ")
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if result, ok := msg.(GitStageResultMsg); !ok {
			t.Errorf("cmd() returned %T, want GitStageResultMsg", msg)
		} else if result.Path != "dot_zshrc" {
			t.Errorf("result.Path = %q, want dot_zshrc", result.Path)
		}
	})

	t.Run("space on fully staged file unstages", func(t *testing.T) {
		m := setupWithEntries()
		m.gitStatus.cursor = 1 // XY="M " (fully staged)
		_, cmd := sendKey(m, " ")
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if result, ok := msg.(GitStageResultMsg); !ok {
			t.Errorf("cmd() returned %T, want GitStageResultMsg", msg)
		} else if result.Path != "dot_bashrc" {
			t.Errorf("result.Path = %q, want dot_bashrc", result.Path)
		}
	})

	t.Run("space on untracked stages", func(t *testing.T) {
		m := setupWithEntries()
		m.gitStatus.cursor = 2 // XY="??" (untracked)
		_, cmd := sendKey(m, " ")
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if _, ok := msg.(GitStageResultMsg); !ok {
			t.Errorf("cmd() returned %T, want GitStageResultMsg", msg)
		}
	})

	t.Run("c opens commit overlay", func(t *testing.T) {
		m := setupWithEntries()
		m, _ = sendKey(m, "c")
		if m.overlay != OverlayCommit {
			t.Errorf("overlay = %d, want OverlayCommit", m.overlay)
		}
	})

	t.Run("P triggers push", func(t *testing.T) {
		m := setupWithEntries()
		_, cmd := sendKey(m, "P")
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if _, ok := msg.(PushResultMsg); !ok {
			t.Errorf("cmd() returned %T, want PushResultMsg", msg)
		}
	})

	t.Run("p triggers pull", func(t *testing.T) {
		m := setupWithEntries()
		_, cmd := sendKey(m, "p")
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if _, ok := msg.(PullResultMsg); !ok {
			t.Errorf("cmd() returned %T, want PullResultMsg", msg)
		}
	})

	t.Run("a triggers stage all", func(t *testing.T) {
		m := setupWithEntries()
		_, cmd := sendKey(m, "a")
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
		msg := cmd()
		if _, ok := msg.(GitStageResultMsg); !ok {
			t.Errorf("cmd() returned %T, want GitStageResultMsg", msg)
		}
	})

	t.Run("D opens discard confirm", func(t *testing.T) {
		m := setupWithEntries()
		m, _ = sendKey(m, "D")
		if m.overlay != OverlayConfirmGitDiscard {
			t.Errorf("overlay = %d, want OverlayConfirmGitDiscard", m.overlay)
		}
		if m.discardPath != "dot_zshrc" {
			t.Errorf("discardPath = %q, want dot_zshrc", m.discardPath)
		}
	})

	t.Run("D on untracked sets discardUntracked", func(t *testing.T) {
		m := setupWithEntries()
		m.gitStatus.cursor = 2 // XY="??"
		m, _ = sendKey(m, "D")
		if !m.discardUntracked {
			t.Error("discardUntracked should be true for ?? entry")
		}
	})
}

// --- Smoke tests ---

func TestViewDoesNotPanic(t *testing.T) {
	t.Run("zero dimensions", func(t *testing.T) {
		m := New(newMockChezmoi(), newMockGit())
		got := m.View()
		if got != "Loading..." {
			t.Errorf("View() = %q, want 'Loading...'", got)
		}
	})

	t.Run("with dimensions and empty state", func(t *testing.T) {
		m, _, _ := newTestModel()
		// Should not panic
		_ = m.View()
	})

	t.Run("with populated state", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.managedFiles = []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		m.statusData = []chezmoi.StatusEntry{
			{SourceState: ' ', DestState: 'M', Path: ".zshrc"},
		}
		m.rebuildFileList()
		m.diffView.SetDimensions(80, 20)
		m.diffView.SetContent(".zshrc", "+new\n-old\n")
		m.updateDimensions()
		// Should not panic
		_ = m.View()
	})

	t.Run("with overlays", func(t *testing.T) {
		m, _, _ := newTestModel()
		overlays := []OverlayMode{OverlayHelp, OverlayCommit, OverlayConfirmApplyAll, OverlayConfirmGitDiscard, OverlayConfirmForget}
		for _, o := range overlays {
			m.overlay = o
			_ = m.View() // Should not panic
		}
	})

	t.Run("with add file overlay", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.addFile = NewAddFileModel([]string{".config/new", ".profile"}, 50, 15)
		m.overlay = OverlayAddFile
		_ = m.View() // Should not panic
	})

	t.Run("with empty add file overlay", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.addFile = NewAddFileModel([]string{}, 50, 15)
		m.overlay = OverlayAddFile
		_ = m.View() // Should not panic
	})

	t.Run("narrow mode", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.width = 60
		m.updateDimensions()
		_ = m.View() // Should not panic
	})
}
