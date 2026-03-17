package ui

import (
	"strings"
	"testing"
)

func TestColorizeDiff(t *testing.T) {
	t.Run("line count preserved", func(t *testing.T) {
		input := "--- a/file\n+++ b/file\n@@ -1,2 +1,2 @@\n-old\n+new\n context\n"
		inputLines := strings.Split(input, "\n")
		outputLines := strings.Split(colorizeDiff(input), "\n")
		if len(outputLines) != len(inputLines) {
			t.Errorf("output has %d lines, input has %d", len(outputLines), len(inputLines))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := colorizeDiff("")
		if got == "" {
			return // acceptable
		}
		// Should not panic — that's the main check
	})

	t.Run("add lines contain original text", func(t *testing.T) {
		got := colorizeDiff("+added line")
		if !strings.Contains(got, "+added line") {
			t.Errorf("output %q does not contain original text", got)
		}
	})

	t.Run("del lines contain original text", func(t *testing.T) {
		got := colorizeDiff("-removed line")
		if !strings.Contains(got, "-removed line") {
			t.Errorf("output %q does not contain original text", got)
		}
	})

	t.Run("hunk lines contain original text", func(t *testing.T) {
		got := colorizeDiff("@@ -1,3 +1,3 @@")
		if !strings.Contains(got, "@@ -1,3 +1,3 @@") {
			t.Errorf("output %q does not contain original text", got)
		}
	})

	t.Run("context lines pass through", func(t *testing.T) {
		got := colorizeDiff(" unchanged")
		if !strings.Contains(got, " unchanged") {
			t.Errorf("output %q does not contain original text", got)
		}
	})
}
