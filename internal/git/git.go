package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type StatusEntry struct {
	XY   string // e.g. "M ", "??", "A "
	Path string
}

type Runner interface {
	Status(ctx context.Context) ([]StatusEntry, error)
	Diff(ctx context.Context, path string) (string, error)
	Add(ctx context.Context, path string) error
	AddAll(ctx context.Context) error
	Commit(ctx context.Context, message string) error
	Push(ctx context.Context) error
	Pull(ctx context.Context) error
	Restore(ctx context.Context, path string) error
	Clean(ctx context.Context, path string) error
}

type CLI struct {
	sourceDir string
}

func NewCLI(sourceDir string) *CLI {
	return &CLI{sourceDir: sourceDir}
}

func (c *CLI) Status(ctx context.Context) ([]StatusEntry, error) {
	out, err := c.run(ctx, "status", "--short")
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	var entries []StatusEntry
	for _, line := range splitLines(out) {
		if len(line) < 4 {
			continue
		}
		entries = append(entries, StatusEntry{
			XY:   line[:2],
			Path: strings.TrimSpace(line[3:]),
		})
	}
	return entries, nil
}

func (c *CLI) Diff(ctx context.Context, path string) (string, error) {
	// Tracked files: show all changes from HEAD (staged + unstaged)
	out, _ := c.run(ctx, "diff", "HEAD", "--", path)
	if out != "" {
		return out, nil
	}
	// Untracked files: generate diff against /dev/null
	out, _ = c.run(ctx, "diff", "--no-index", "--", "/dev/null", path)
	if out != "" {
		return out, nil
	}
	return "", nil
}

func (c *CLI) Add(ctx context.Context, path string) error {
	_, err := c.run(ctx, "add", "--", path)
	if err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	return nil
}

func (c *CLI) AddAll(ctx context.Context) error {
	_, err := c.run(ctx, "add", "-A")
	if err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	return nil
}

func (c *CLI) Commit(ctx context.Context, message string) error {
	_, err := c.run(ctx, "commit", "-m", message)
	if err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

func (c *CLI) Push(ctx context.Context) error {
	_, err := c.run(ctx, "push")
	if err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

func (c *CLI) Pull(ctx context.Context) error {
	_, err := c.run(ctx, "pull")
	if err != nil {
		return fmt.Errorf("git pull: %w", err)
	}
	return nil
}

func (c *CLI) Restore(ctx context.Context, path string) error {
	_, err := c.run(ctx, "checkout", "HEAD", "--", path)
	if err != nil {
		return fmt.Errorf("git restore: %w", err)
	}
	return nil
}

func (c *CLI) Clean(ctx context.Context, path string) error {
	_, err := c.run(ctx, "clean", "-f", "--", path)
	if err != nil {
		return fmt.Errorf("git clean: %w", err)
	}
	return nil
}

func (c *CLI) run(ctx context.Context, args ...string) (string, error) {
	fullArgs := append([]string{"-C", c.sourceDir}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
