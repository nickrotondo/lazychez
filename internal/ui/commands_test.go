package ui

import "testing"

func TestReverseDiff(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "swap add and del lines",
			input: "+added\n-removed\n context\n",
			want:  "-added\n+removed\n context\n",
		},
		{
			name:  "swap header lines",
			input: "--- a/file\n+++ b/file\n",
			want:  "+++ a/file\n--- b/file\n",
		},
		{
			name:  "preserve hunk headers",
			input: "@@ -1,3 +1,3 @@\n",
			want:  "@@ -1,3 +1,3 @@\n",
		},
		{
			name:  "preserve context lines",
			input: " context line\n",
			want:  " context line\n",
		},
		{
			name: "full unified diff",
			input: "--- a/file\n+++ b/file\n@@ -1,3 +1,3 @@\n context\n-old\n+new\n",
			want:  "+++ a/file\n--- b/file\n@@ -1,3 +1,3 @@\n context\n+old\n-new\n",
		},
		{
			name:  "trailing newline preserved",
			input: "+line\n",
			want:  "-line\n",
		},
		{
			name:  "no trailing newline preserved",
			input: "+line",
			want:  "-line",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "diff meta line preserved",
			input: "diff --git a/f b/f\n",
			want:  "diff --git a/f b/f\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reverseDiff(tt.input)
			if got != tt.want {
				t.Errorf("reverseDiff() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}
