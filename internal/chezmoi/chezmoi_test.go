package chezmoi

import "testing"

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty string", "", nil},
		{"single line", "one", []string{"one"}},
		{"two lines", "one\ntwo", []string{"one", "two"}},
		{"trailing newline trimmed", "one\ntwo\n", []string{"one", "two"}},
		{"multiple trailing newlines trimmed", "one\ntwo\n\n", []string{"one", "two"}},
		{"only newlines", "\n", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitLines(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitLines(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
