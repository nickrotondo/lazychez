package ui

import "testing"

func TestIsGUIEditor(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"code", true},
		{"code-insiders", true},
		{"cursor", true},
		{"subl", true},
		{"zed", true},
		{"goland", true},
		{"webstorm", true},
		{"vim", false},
		{"nvim", false},
		{"nano", false},
		{"emacs", false},
		{"vi", false},
		{"/usr/local/bin/code", true},
		{"/opt/homebrew/bin/zed", true},
		{"/usr/bin/vim", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			if got := isGUIEditor(tt.command); got != tt.want {
				t.Errorf("isGUIEditor(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}
