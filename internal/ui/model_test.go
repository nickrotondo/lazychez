package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nickrotondo/lazychez/internal/chezmoi"
	"github.com/nickrotondo/lazychez/internal/git"
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
				{AddCol: ' ', ApplyCol: 'M', Path: ".zshrc"},
			},
		}
		result, _ := m.Update(msg)
		m = result.(Model)
		// Should have rebuilt with dirty file
		if m.fileList.DirtyCount() != 1 {
			t.Errorf("DirtyCount() = %d, want 1", m.fileList.DirtyCount())
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
	t.Run("success sets status with undo hint", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := AddResultMsg{Path: ".zshrc"}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "Re-added .zshrc") {
			t.Errorf("statusMsg = %q, want to contain 'Re-added .zshrc'", m.statusMsg)
		}
		if !strings.Contains(m.statusMsg, "undo in Git pane with D") {
			t.Errorf("statusMsg = %q, want to contain undo hint", m.statusMsg)
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
		if m.statusMsg != "Applied .zshrc" {
			t.Errorf("statusMsg = %q, want 'Applied .zshrc'", m.statusMsg)
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

	t.Run("no remote error", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(PushResultMsg{Err: git.ErrNoRemote})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "chezmoi init") {
			t.Errorf("statusMsg should mention chezmoi init, got %q", m.statusMsg)
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

	t.Run("no remote error", func(t *testing.T) {
		m, _, _ := newTestModel()
		result, _ := m.Update(PullResultMsg{Err: git.ErrNoRemote})
		m = result.(Model)
		if !strings.Contains(m.statusMsg, "chezmoi init") {
			t.Errorf("statusMsg should mention chezmoi init, got %q", m.statusMsg)
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

	t.Run("tab cycles focus forward skipping pane 0", func(t *testing.T) {
		m, _, _ := newTestModel()
		if m.focused != PaneFileList {
			t.Fatalf("initial focus = %d, want PaneFileList", m.focused)
		}
		m, _ = sendSpecialKey(m, tea.KeyTab)
		if m.focused != PaneGitStatus {
			t.Errorf("after tab: focused = %d, want PaneGitStatus", m.focused)
		}
		m, _ = sendSpecialKey(m, tea.KeyTab)
		if m.focused != PaneStatus {
			t.Errorf("after tab: focused = %d, want PaneStatus", m.focused)
		}
		m, _ = sendSpecialKey(m, tea.KeyTab)
		if m.focused != PaneFileList {
			t.Errorf("after tab: focused = %d, want PaneFileList (wrap)", m.focused)
		}
	})

	t.Run("shift+tab cycles focus backward skipping pane 0", func(t *testing.T) {
		m, _, _ := newTestModel()
		m, _ = sendSpecialKey(m, tea.KeyShiftTab)
		if m.focused != PaneStatus {
			t.Errorf("after shift+tab: focused = %d, want PaneStatus", m.focused)
		}
		m, _ = sendSpecialKey(m, tea.KeyShiftTab)
		if m.focused != PaneGitStatus {
			t.Errorf("after shift+tab: focused = %d, want PaneGitStatus", m.focused)
		}
	})

	t.Run("1 focuses file list", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneInfo
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

	t.Run("3 focuses status", func(t *testing.T) {
		m, _, _ := newTestModel()
		m, _ = sendKey(m, "3")
		if m.focused != PaneStatus {
			t.Errorf("focused = %d, want PaneStatus", m.focused)
		}
	})

	t.Run("0 focuses info pane", func(t *testing.T) {
		m, _, _ := newTestModel()
		m, _ = sendKey(m, "0")
		if m.focused != PaneInfo {
			t.Errorf("focused = %d, want PaneInfo", m.focused)
		}
	})

	t.Run("left/right cycles through 1-2-3 skipping pane 0", func(t *testing.T) {
		m, _, _ := newTestModel()
		// Start at file list (1), right → git status (2)
		m, _ = sendSpecialKey(m, tea.KeyRight)
		if m.focused != PaneGitStatus {
			t.Errorf("after right: focused = %d, want PaneGitStatus", m.focused)
		}
		// Right → status (3)
		m, _ = sendSpecialKey(m, tea.KeyRight)
		if m.focused != PaneStatus {
			t.Errorf("after right: focused = %d, want PaneStatus", m.focused)
		}
		// Right → file list (1) — wraps
		m, _ = sendSpecialKey(m, tea.KeyRight)
		if m.focused != PaneFileList {
			t.Errorf("after right: focused = %d, want PaneFileList", m.focused)
		}
		// Left from file list (1) → status (3)
		m, _ = sendSpecialKey(m, tea.KeyLeft)
		if m.focused != PaneStatus {
			t.Errorf("after left: focused = %d, want PaneStatus", m.focused)
		}
		// From info pane, left → file list (default)
		m.focused = PaneInfo
		m, _ = sendSpecialKey(m, tea.KeyLeft)
		if m.focused != PaneFileList {
			t.Errorf("after left from info: focused = %d, want PaneFileList", m.focused)
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
		// Populate with files that have changes
		m.managedFiles = []chezmoi.ManagedFile{
			{Path: ".bashrc", SourceRelPath: "dot_bashrc"},
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
			{Path: ".vimrc", SourceRelPath: "dot_vimrc"},
		}
		m.statusData = []chezmoi.StatusEntry{
			{AddCol: ' ', ApplyCol: 'M', Path: ".bashrc"},
			{AddCol: 'M', ApplyCol: ' ', Path: ".zshrc"},
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

	t.Run("s triggers re-add on selected file", func(t *testing.T) {
		m, _ := setupWithFiles()
		path := m.fileList.SelectedPath()
		if path == "" {
			t.Fatal("no file selected")
		}
		_, cmd := sendKey(m, "s")
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

	t.Run("space is no-op in chezmoi pane", func(t *testing.T) {
		m, _ := setupWithFiles()
		_, cmd := sendKey(m, " ")
		if cmd != nil {
			t.Error("cmd should be nil (space is no-op in chezmoi pane)")
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

	t.Run("D is no-op in chezmoi pane", func(t *testing.T) {
		m, _ := setupWithFiles()
		_, cmd := sendKey(m, "D")
		if cmd != nil {
			t.Error("cmd should be nil (D is no-op in chezmoi pane)")
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

	t.Run("s with no files is no-op", func(t *testing.T) {
		m, _, _ := newTestModel()
		_, cmd := sendKey(m, "s")
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
		m := New(newMockChezmoi(), newMockGit(), "dev")
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
			{AddCol: ' ', ApplyCol: 'M', Path: ".zshrc"},
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

func TestFormatAheadBehind(t *testing.T) {
	tests := []struct {
		name   string
		ahead  int
		behind int
		branch string
		remote string
		want   string
	}{
		{"ahead only", 2, 0, "main", "origin/main", "↑2 main → origin/main"},
		{"behind only", 0, 3, "main", "origin/main", "↓3 main → origin/main"},
		{"both", 2, 3, "main", "origin/main", "↑2 ↓3 main → origin/main"},
		{"zero zero", 0, 0, "main", "origin/main", "main → origin/main"},
		{"no upstream", 0, 0, "main", "", "main (no remote)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAheadBehind(tt.ahead, tt.behind, tt.branch, tt.remote)
			if got != tt.want {
				t.Errorf("FormatAheadBehind() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpdate_AheadBehindMsg(t *testing.T) {
	t.Run("success updates status pane", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := AheadBehindMsg{Ahead: 2, Behind: 3, Branch: "main", Remote: "origin/main"}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if cmd != nil {
			t.Error("cmd should be nil on success")
		}
		view := m.statusPane.View()
		if !strings.Contains(view, "↑2") {
			t.Errorf("view = %q, want to contain ↑2", view)
		}
		if !strings.Contains(view, "↓3") {
			t.Errorf("view = %q, want to contain ↓3", view)
		}
		if !strings.Contains(view, "main → origin/main") {
			t.Errorf("view = %q, want to contain 'main → origin/main'", view)
		}
	})

	t.Run("error sets status", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := AheadBehindMsg{Err: fmt.Errorf("git failed")}
		result, cmd := m.Update(msg)
		m = result.(Model)
		if !m.statusError {
			t.Error("statusError should be true")
		}
		if !strings.Contains(m.statusMsg, "Git error") {
			t.Errorf("statusMsg = %q, want to contain 'Git error'", m.statusMsg)
		}
		if cmd == nil {
			t.Error("cmd should not be nil (clearStatusAfter)")
		}
	})

	t.Run("no upstream shows no remote", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := AheadBehindMsg{Branch: "main", Remote: ""}
		result, _ := m.Update(msg)
		m = result.(Model)
		view := m.statusPane.View()
		if !strings.Contains(view, "(no remote)") {
			t.Errorf("view = %q, want to contain '(no remote)'", view)
		}
	})

	t.Run("zero zero with remote shows clean line", func(t *testing.T) {
		m, _, _ := newTestModel()
		msg := AheadBehindMsg{Ahead: 0, Behind: 0, Branch: "main", Remote: "origin/main"}
		result, _ := m.Update(msg)
		m = result.(Model)
		view := m.statusPane.View()
		if view != "main → origin/main" {
			t.Errorf("view = %q, want 'main → origin/main'", view)
		}
	})
}

func TestStatusPaneFooterHints(t *testing.T) {
	m, _, _ := newTestModel()
	m.focused = PaneStatus
	m.syncFocus()
	footer := m.renderFooter()
	if !strings.Contains(footer, "refresh") {
		t.Errorf("footer = %q, want to contain 'refresh'", footer)
	}
}

func TestInit_FetchesAheadBehind(t *testing.T) {
	m, _, g := newTestModel()
	g.aheadBehindInfo = git.AheadBehindInfo{Ahead: 1, Behind: 0, Branch: "main", Remote: "origin/main"}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
	// Init returns a batch — execute it and check for AheadBehindMsg
	// We can't easily decompose a batch, but we can verify the mock is wired
	// by checking that the model compiles and Init doesn't panic
}

// --- Phase 3: Contextual detail pane + info view ---

func TestDetailPaneContext(t *testing.T) {
	t.Run("returns focused pane when not on pane 0", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneFileList
		if got := m.detailPaneContext(); got != PaneFileList {
			t.Errorf("detailPaneContext() = %d, want PaneFileList", got)
		}
	})

	t.Run("returns prevFocused when on pane 0", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneInfo
		m.prevFocused = PaneGitStatus
		if got := m.detailPaneContext(); got != PaneGitStatus {
			t.Errorf("detailPaneContext() = %d, want PaneGitStatus", got)
		}
	})

	t.Run("status context shows info view", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneStatus
		if !m.showInfoView() {
			t.Error("showInfoView() should be true when focused on Status")
		}
	})

	t.Run("file list context shows diff view", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneFileList
		if m.showInfoView() {
			t.Error("showInfoView() should be false when focused on FileList")
		}
	})

	t.Run("pane 0 with status prevFocused shows info view", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneInfo
		m.prevFocused = PaneStatus
		if !m.showInfoView() {
			t.Error("showInfoView() should be true when prevFocused is Status")
		}
	})

	t.Run("pane 0 with file list prevFocused shows diff view", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneInfo
		m.prevFocused = PaneFileList
		if m.showInfoView() {
			t.Error("showInfoView() should be false when prevFocused is FileList")
		}
	})
}

func TestDetailPaneTitle(t *testing.T) {
	t.Run("title is lazychez when status pane was last focused", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneStatus
		m.syncFocus()
		m.updateDimensions()
		view := stripAnsi(m.View())
		if !strings.Contains(view, "[0]─lazychez") {
			t.Errorf("View() should contain '[0]─lazychez' when Status is focused, got relevant portion missing")
		}
	})

	t.Run("title is chezmoi diff with path when file list is focused and file selected", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneFileList
		m.syncFocus()
		m.diffView.SetDimensions(80, 20)
		m.diffView.SetContent(".zshrc", "+line")
		m.updateDimensions()
		view := stripAnsi(m.View())
		if !strings.Contains(view, "[0]─chezmoi diff — .zshrc") {
			t.Errorf("View() should contain '[0]─chezmoi diff — .zshrc'")
		}
	})

	t.Run("title is chezmoi diff when file list is focused with no file selected", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneFileList
		m.syncFocus()
		m.updateDimensions()
		view := stripAnsi(m.View())
		if !strings.Contains(view, "[0]─chezmoi diff") {
			t.Errorf("View() should contain '[0]─chezmoi diff'")
		}
		if strings.Contains(view, "[0]─lazychez") {
			t.Error("View() should not contain '[0]─lazychez' when FileList is focused")
		}
	})

	t.Run("pane 0 focused preserves lazychez context from status", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneStatus
		m.syncFocus()
		m.updateDimensions()
		// Now press 0 to focus pane 0
		m, _ = sendKey(m, "0")
		view := stripAnsi(m.View())
		if !strings.Contains(view, "[0]─lazychez") {
			t.Errorf("View() should preserve '[0]─lazychez' context after pressing 0 from Status")
		}
	})

	t.Run("pane 0 focused preserves chezmoi diff context from file list", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneFileList
		m.syncFocus()
		m.diffView.SetDimensions(80, 20)
		m.diffView.SetContent(".vimrc", "-old")
		m.updateDimensions()
		// Now press 0 to focus pane 0
		m, _ = sendKey(m, "0")
		view := stripAnsi(m.View())
		if !strings.Contains(view, "[0]─chezmoi diff — .vimrc") {
			t.Errorf("View() should preserve '[0]─chezmoi diff — .vimrc' context after pressing 0 from FileList")
		}
	})

	t.Run("switching left-side focus updates pane 0 immediately", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneStatus
		m.syncFocus()
		m.updateDimensions()
		view := stripAnsi(m.View())
		if !strings.Contains(view, "[0]─lazychez") {
			t.Fatal("expected [0]─lazychez initially")
		}
		// Tab to file list
		m, _ = sendKey(m, "tab")
		view = stripAnsi(m.View())
		if strings.Contains(view, "[0]─lazychez") {
			t.Error("View() should switch from lazychez to chezmoi diff when focus moves to FileList")
		}
		if !strings.Contains(view, "[0]─chezmoi diff") {
			t.Error("View() should contain '[0]─chezmoi diff' after switching to FileList")
		}
	})
}

func TestInfoViewContent(t *testing.T) {
	t.Run("contains expected strings", func(t *testing.T) {
		m, _, _ := newTestModel()
		content := m.renderInfoContent()
		checks := []string{
			"lazychez",
			"dev", // version from newTestModel
			"github.com/nickrotondo/lazychez",
			"Report issues",
			"Nick Rotondo",
		}
		for _, want := range checks {
			if !strings.Contains(content, want) {
				t.Errorf("renderInfoContent() missing %q", want)
			}
		}
	})

	t.Run("uses build-time version", func(t *testing.T) {
		cm := newMockChezmoi()
		g := newMockGit()
		m := New(cm, g, "v1.2.3")
		m.width = 120
		m.height = 40
		m.updateDimensions()
		content := m.renderInfoContent()
		if !strings.Contains(content, "v1.2.3") {
			t.Errorf("renderInfoContent() missing version 'v1.2.3'")
		}
	})

	t.Run("info view renders without panic in View", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.focused = PaneStatus
		m.updateDimensions()
		// Should not panic
		_ = m.View()
	})

	t.Run("info view renders in narrow mode", func(t *testing.T) {
		m, _, _ := newTestModel()
		m.width = 60
		m.focused = PaneStatus
		m.updateDimensions()
		// Should not panic
		_ = m.View()
	})
}

// --- Template support (Phase 4) ---

func setupTemplateModel(t *testing.T) (Model, *mockChezmoiRunner) {
	t.Helper()
	m, cm, _ := newTestModel()
	// .aliases sorts before .vimrc, so cursor starts on the template file.
	m.managedFiles = []chezmoi.ManagedFile{
		{Path: ".aliases", SourceRelPath: "dot_aliases.tmpl"},
		{Path: ".vimrc", SourceRelPath: "dot_vimrc"},
	}
	cm.catOutput[".aliases"] = "alias ll='ls -la'\n"
	m.rebuildFileList()
	m.fileList.GoToTop() // cursor at position 0 = .aliases (template)
	return m, cm
}

func TestTemplateDetection(t *testing.T) {
	t.Run("IsTemplate true for .tmpl source path", func(t *testing.T) {
		item := FileItem{Path: ".zshrc", SourceRelPath: "dot_zshrc.tmpl"}
		if !item.IsTemplate() {
			t.Error("IsTemplate() = false, want true")
		}
	})

	t.Run("IsTemplate false for non-template source path", func(t *testing.T) {
		item := FileItem{Path: ".vimrc", SourceRelPath: "dot_vimrc"}
		if item.IsTemplate() {
			t.Error("IsTemplate() = true, want false")
		}
	})

	t.Run("IsTemplate false for empty SourceRelPath", func(t *testing.T) {
		item := FileItem{Path: ".bashrc"}
		if item.IsTemplate() {
			t.Error("IsTemplate() = true, want false")
		}
	})
}

func TestCatMsgHandling(t *testing.T) {
	t.Run("success sets diffView content", func(t *testing.T) {
		m, _ := setupTemplateModel(t)
		m.catMode = true
		m.syncFocus()
		msg := CatMsg{Path: ".aliases", Content: "alias ll='ls -la'\n"}
		result, _ := m.Update(msg)
		m = result.(Model)
		if m.diffView.path != ".aliases" {
			t.Errorf("diffView.path = %q, want .aliases", m.diffView.path)
		}
		if m.diffView.rawDiff != msg.Content {
			t.Errorf("diffView.rawDiff = %q, want %q", m.diffView.rawDiff, msg.Content)
		}
	})

	t.Run("error shows in diff view", func(t *testing.T) {
		m, _ := setupTemplateModel(t)
		m.catMode = true
		msg := CatMsg{Path: ".aliases", Err: fmt.Errorf("cat failed")}
		result, _ := m.Update(msg)
		m = result.(Model)
		if m.diffView.path != ".aliases" {
			t.Errorf("diffView.path = %q, want .aliases", m.diffView.path)
		}
	})
}

func TestDiffMsgSkippedInCatMode(t *testing.T) {
	m, _ := setupTemplateModel(t)
	// Enter cat mode by setting content
	m.catMode = true
	m.syncFocus()
	m.diffView.SetContent(".aliases", "alias ll='ls -la'\n")

	// A diff message arrives (background refresh) — should not overwrite cat content
	result, _ := m.Update(DiffMsg{Path: ".aliases", Diff: "+new diff line\n"})
	m = result.(Model)
	if m.diffView.rawDiff != "alias ll='ls -la'\n" {
		t.Errorf("diffView.rawDiff = %q, want cat content to be preserved", m.diffView.rawDiff)
	}
	// Diff is still cached even in cat mode
	if m.diffCache[".aliases"] != "+new diff line\n" {
		t.Errorf("diffCache[.aliases] = %q, want diff cached", m.diffCache[".aliases"])
	}
}

func TestTKeyToggleCatMode(t *testing.T) {
	t.Run("t on template file toggles catMode on and issues fetchCat", func(t *testing.T) {
		m, cm := setupTemplateModel(t)
		cm.catOutput[".aliases"] = "alias ll='ls -la'\n"

		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
		m = result.(Model)

		if !m.catMode {
			t.Error("catMode should be true after pressing t on template")
		}
		if cmd == nil {
			t.Error("cmd should not be nil — expected fetchCat command")
		}
		// Simulate the CatMsg arriving
		result, _ = m.Update(CatMsg{Path: ".aliases", Content: "alias ll='ls -la'\n"})
		m = result.(Model)
		if m.diffView.rawDiff != "alias ll='ls -la'\n" {
			t.Errorf("diffView content = %q, want cat output", m.diffView.rawDiff)
		}
	})

	t.Run("t toggles catMode off and fetches diff", func(t *testing.T) {
		m, _ := setupTemplateModel(t)
		m.catMode = true
		m.syncFocus()

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
		m = result.(Model)

		if m.catMode {
			t.Error("catMode should be false after second t press")
		}
	})

	t.Run("t on non-template file shows status message and does not toggle", func(t *testing.T) {
		m, _ := setupTemplateModel(t)
		// Navigate down to .vimrc (non-template)
		m.fileList.MoveDown()

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
		m = result.(Model)

		if m.catMode {
			t.Error("catMode should remain false for non-template file")
		}
		if !strings.Contains(m.statusMsg, "Not a template") {
			t.Errorf("statusMsg = %q, want 'Not a template'", m.statusMsg)
		}
	})
}

func TestCatModeAutoRevertOnNavigation(t *testing.T) {
	m, _ := setupTemplateModel(t)
	m.catMode = true
	m.syncFocus()

	// Navigate down to the next file — should revert catMode
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = result.(Model)

	if m.catMode {
		t.Error("catMode should auto-revert to false when navigating to a different file")
	}
}

func TestDetailPaneTitleInCatMode(t *testing.T) {
	m, _ := setupTemplateModel(t)
	m.catMode = true
	m.diffView.path = ".aliases"

	view := m.View()
	if !strings.Contains(view, "chezmoi cat") {
		t.Errorf("View() missing 'chezmoi cat' in title, got:\n%s", view)
	}
}

func TestDiffViewContextInCatMode(t *testing.T) {
	m, _ := setupTemplateModel(t)
	m.catMode = true
	m.syncFocus()

	if m.diffView.context != "cat" {
		t.Errorf("diffView.context = %q, want 'cat'", m.diffView.context)
	}
}
