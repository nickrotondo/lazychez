package ui

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// GUI editors that don't need terminal control — run them async so the TUI stays visible.
var guiEditors = map[string]bool{
	"code": true, "code-insiders": true,
	"cursor": true,
	"subl": true, "sublime_text": true,
	"zed": true,
	"atom": true,
	"fleet": true,
	"idea": true, "goland": true, "webstorm": true, "pycharm": true,
}

func isGUIEditor(command string) bool {
	base := filepath.Base(command)
	return guiEditors[base]
}

func resolveEditor() (string, []string) {
	out, err := exec.Command("chezmoi", "dump-config", "--format=json").Output()
	if err == nil {
		var cfg struct {
			Edit struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			} `json:"edit"`
		}
		if json.Unmarshal(out, &cfg) == nil && cfg.Edit.Command != "" {
			return cfg.Edit.Command, cfg.Edit.Args
		}
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor, nil
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, nil
	}
	return "vi", nil
}

func editorCmd(editor string, args []string, filePath string) tea.Cmd {
	fullArgs := append(append([]string{}, args...), filePath)
	if isGUIEditor(editor) {
		return func() tea.Msg {
			err := exec.Command(editor, fullArgs...).Run()
			return EditorFinishedMsg{Err: err}
		}
	}
	c := exec.Command(editor, fullArgs...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func openInEditor(filePath string) tea.Cmd {
	editor, args := resolveEditor()
	return editorCmd(editor, args, filePath)
}

func chezmoiEdit(filePath string) tea.Cmd {
	editor, _ := resolveEditor()
	if isGUIEditor(editor) {
		return func() tea.Msg {
			err := exec.Command("chezmoi", "edit", filePath).Run()
			return EditorFinishedMsg{Err: err}
		}
	}
	c := exec.Command("chezmoi", "edit", filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func chezmoiEditConfig() tea.Cmd {
	editor, _ := resolveEditor()
	if isGUIEditor(editor) {
		return func() tea.Msg {
			err := exec.Command("chezmoi", "edit-config").Run()
			return EditorFinishedMsg{Err: err}
		}
	}
	c := exec.Command("chezmoi", "edit-config")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}
