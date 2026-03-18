package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNoRemote indicates that no git remote is configured for push/pull.
var ErrNoRemote = errors.New("no remote configured — run 'chezmoi init <repo-url>' to set one up")

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
	Reset(ctx context.Context, path string) error
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
	out, err := c.run(ctx, "diff", "HEAD", "--", path)
	if out != "" {
		return out, nil
	}
	if err != nil && !isExitCode1(err) {
		return "", fmt.Errorf("git diff: %w", err)
	}
	// Untracked files: generate diff against /dev/null
	out, err = c.run(ctx, "diff", "--no-index", "--", "/dev/null", path)
	if out != "" {
		return out, nil
	}
	if err != nil && !isExitCode1(err) {
		return "", fmt.Errorf("git diff --no-index: %w", err)
	}
	return "", nil
}

// isExitCode1 returns true if the error is an exec.ExitError with code 1,
// which git uses to indicate "files differ" (not a real error).
func isExitCode1(err error) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() == 1
	}
	return false
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
		if isNoRemoteErr(err) {
			return ErrNoRemote
		}
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

func (c *CLI) Pull(ctx context.Context) error {
	_, err := c.run(ctx, "pull")
	if err != nil {
		if isNoRemoteErr(err) {
			return ErrNoRemote
		}
		return fmt.Errorf("git pull: %w", err)
	}
	return nil
}

func isNoRemoteErr(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no configured push destination") ||
		strings.Contains(msg, "no tracking information") ||
		strings.Contains(msg, "no such remote")
}

func (c *CLI) Reset(ctx context.Context, path string) error {
	_, err := c.run(ctx, "reset", "HEAD", "--", path)
	if err != nil {
		return fmt.Errorf("git reset: %w", err)
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
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return string(out), fmt.Errorf("%s", strings.TrimSpace(string(ee.Stderr)))
		}
	}
	return string(out), err
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
