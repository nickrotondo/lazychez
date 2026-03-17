package chezmoi

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ManagedFile struct {
	Path          string // relative to home, e.g. ".zshrc"
	SourceRelPath string // relative to source dir, e.g. "dot_zshrc"
}

type StatusEntry struct {
	SourceState rune   // 'M', 'A', 'D', ' '
	DestState   rune   // 'M', 'A', 'D', ' '
	Path        string // relative to home
}

type Runner interface {
	Managed(ctx context.Context) ([]ManagedFile, error)
	Status(ctx context.Context) ([]StatusEntry, error)
	Diff(ctx context.Context, path string) (string, error)
	Add(ctx context.Context, path string) error
	Apply(ctx context.Context, path string) error
	ApplyAll(ctx context.Context) error
	Forget(ctx context.Context, path string) error
	SourcePath() string
}

type CLI struct {
	sourcePath string
	homeDir    string
}

func NewCLI() (*CLI, error) {
	out, err := exec.Command("chezmoi", "source-path").Output()
	if err != nil {
		return nil, fmt.Errorf("chezmoi source-path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("user home dir: %w", err)
	}

	return &CLI{
		sourcePath: strings.TrimSpace(string(out)),
		homeDir:    home,
	}, nil
}

func (c *CLI) SourcePath() string {
	return c.sourcePath
}

func (c *CLI) Managed(ctx context.Context) ([]ManagedFile, error) {
	out, err := c.run(ctx, "managed", "--include=files")
	if err != nil {
		return nil, fmt.Errorf("chezmoi managed: %w", err)
	}

	srcOut, err := c.run(ctx, "managed", "--include=files", "--path-style=source-relative")
	if err != nil {
		return nil, fmt.Errorf("chezmoi managed source-relative: %w", err)
	}

	targetLines := splitLines(out)
	srcLines := splitLines(srcOut)

	var files []ManagedFile
	for i, line := range targetLines {
		if line == "" {
			continue
		}
		f := ManagedFile{Path: line}
		if i < len(srcLines) {
			f.SourceRelPath = srcLines[i]
		}
		files = append(files, f)
	}
	return files, nil
}

func (c *CLI) Status(ctx context.Context) ([]StatusEntry, error) {
	out, err := c.run(ctx, "status")
	if err != nil {
		return nil, fmt.Errorf("chezmoi status: %w", err)
	}

	var entries []StatusEntry
	for _, line := range splitLines(out) {
		if len(line) < 3 {
			continue
		}
		entries = append(entries, StatusEntry{
			SourceState: rune(line[0]),
			DestState:   rune(line[1]),
			Path:        strings.TrimSpace(line[2:]),
		})
	}
	return entries, nil
}

func (c *CLI) Diff(ctx context.Context, path string) (string, error) {
	fullPath := filepath.Join(c.homeDir, path)
	out, err := c.run(ctx, "diff", "--", fullPath)
	if err != nil {
		// chezmoi diff exits non-zero when there are differences
		if out != "" {
			return out, nil
		}
		return "", fmt.Errorf("chezmoi diff %s: %w", path, err)
	}
	return out, nil
}

// TemplateEditError indicates that a template file could not be auto-patched.
// The UI should direct the user to chezmoi edit instead.
type TemplateEditError struct {
	Path string
}

func (e *TemplateEditError) Error() string {
	return fmt.Sprintf("%s is a template — use chezmoi edit", e.Path)
}

func (c *CLI) Add(ctx context.Context, path string) error {
	fullPath := filepath.Join(c.homeDir, path)

	srcPath, err := c.runStdout(ctx, "source-path", fullPath)
	if err != nil {
		return fmt.Errorf("chezmoi source-path %s: %w", path, err)
	}
	srcPath = strings.TrimSpace(srcPath)

	if !strings.HasSuffix(srcPath, ".tmpl") {
		_, err := c.run(ctx, "re-add", fullPath)
		if err != nil {
			return fmt.Errorf("chezmoi re-add %s: %w", path, err)
		}
		return nil
	}

	return c.patchTemplate(ctx, path, fullPath, srcPath)
}

func (c *CLI) patchTemplate(ctx context.Context, path, destPath, srcTemplatePath string) error {
	rendered, err := c.runStdout(ctx, "cat", destPath)
	if err != nil {
		return fmt.Errorf("chezmoi cat %s: %w", path, err)
	}

	renderedFile, err := os.CreateTemp("", "chezmoi-rendered-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(renderedFile.Name())

	if _, err := renderedFile.WriteString(rendered); err != nil {
		renderedFile.Close()
		return fmt.Errorf("write rendered temp: %w", err)
	}
	renderedFile.Close()

	// diff exits 1 when files differ (expected), 2+ on error
	diffCmd := exec.CommandContext(ctx, "diff", "-u", renderedFile.Name(), destPath)
	diffOut, diffErr := diffCmd.Output()
	if diffErr != nil {
		if exitErr, ok := diffErr.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// expected — files differ
		} else if len(diffOut) == 0 {
			return fmt.Errorf("diff %s: %w", path, diffErr)
		}
	}

	if len(diffOut) == 0 {
		return nil
	}

	patchedFile, err := os.CreateTemp("", "chezmoi-patched-*")
	if err != nil {
		return fmt.Errorf("create patched temp: %w", err)
	}
	defer os.Remove(patchedFile.Name())
	patchedFile.Close()

	patchCmd := exec.CommandContext(ctx, "patch", "-o", patchedFile.Name(), "--batch", srcTemplatePath)
	patchCmd.Stdin = bytes.NewReader(diffOut)
	if err := patchCmd.Run(); err != nil {
		return &TemplateEditError{Path: path}
	}

	info, err := os.Stat(srcTemplatePath)
	if err != nil {
		return fmt.Errorf("stat source template: %w", err)
	}

	patched, err := os.ReadFile(patchedFile.Name())
	if err != nil {
		return fmt.Errorf("read patched output: %w", err)
	}

	if err := os.WriteFile(srcTemplatePath, patched, info.Mode()); err != nil {
		return fmt.Errorf("write source template: %w", err)
	}

	return nil
}

func (c *CLI) Forget(ctx context.Context, path string) error {
	fullPath := filepath.Join(c.homeDir, path)
	_, err := c.run(ctx, "forget", "--force", fullPath)
	if err != nil {
		return fmt.Errorf("chezmoi forget %s: %w", path, err)
	}
	return nil
}

func (c *CLI) Apply(ctx context.Context, path string) error {
	fullPath := filepath.Join(c.homeDir, path)
	_, err := c.run(ctx, "apply", "--force", fullPath)
	if err != nil {
		return fmt.Errorf("chezmoi apply %s: %w", path, err)
	}
	return nil
}

func (c *CLI) ApplyAll(ctx context.Context) error {
	_, err := c.run(ctx, "apply", "--force")
	if err != nil {
		return fmt.Errorf("chezmoi apply all: %w", err)
	}
	return nil
}

func (c *CLI) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "chezmoi", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (c *CLI) runStdout(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "chezmoi", args...)
	out, err := cmd.Output()
	return string(out), err
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
