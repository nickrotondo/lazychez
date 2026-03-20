package ui

import (
	"context"

	"github.com/nickrotondo/lazychez/internal/chezmoi"
	"github.com/nickrotondo/lazychez/internal/git"
)

// mockChezmoiRunner implements chezmoi.Runner for testing.
type mockChezmoiRunner struct {
	managedFiles    []chezmoi.ManagedFile
	managedErr      error
	unmanagedFiles  []string
	unmanagedErr    error
	statusEntries   []chezmoi.StatusEntry
	statusErr       error
	diffOutput      map[string]string
	diffErr         map[string]error
	catOutput       map[string]string
	catErr          map[string]error
	addErr          map[string]error
	addNewErr       map[string]error
	applyErr        map[string]error
	applyAllErr     error
	forgetErr       map[string]error
	sourcePath      string

	addCalls       []string
	addNewCalls    []string
	applyCalls     []string
	applyAllCalled bool
	forgetCalls    []string
	catCalls       []string
}

func newMockChezmoi() *mockChezmoiRunner {
	return &mockChezmoiRunner{
		diffOutput: make(map[string]string),
		diffErr:    make(map[string]error),
		catOutput:  make(map[string]string),
		catErr:     make(map[string]error),
		addErr:     make(map[string]error),
		addNewErr:  make(map[string]error),
		applyErr:   make(map[string]error),
		forgetErr:  make(map[string]error),
		sourcePath: "/home/user/.local/share/chezmoi",
	}
}

func (m *mockChezmoiRunner) Managed(_ context.Context) ([]chezmoi.ManagedFile, error) {
	return m.managedFiles, m.managedErr
}

func (m *mockChezmoiRunner) Unmanaged(_ context.Context) ([]string, error) {
	return m.unmanagedFiles, m.unmanagedErr
}

func (m *mockChezmoiRunner) Status(_ context.Context) ([]chezmoi.StatusEntry, error) {
	return m.statusEntries, m.statusErr
}

func (m *mockChezmoiRunner) Diff(_ context.Context, path string) (string, error) {
	return m.diffOutput[path], m.diffErr[path]
}

func (m *mockChezmoiRunner) Cat(_ context.Context, path string) (string, error) {
	m.catCalls = append(m.catCalls, path)
	return m.catOutput[path], m.catErr[path]
}

func (m *mockChezmoiRunner) Add(_ context.Context, path string) error {
	m.addCalls = append(m.addCalls, path)
	return m.addErr[path]
}

func (m *mockChezmoiRunner) AddNew(_ context.Context, path string) error {
	m.addNewCalls = append(m.addNewCalls, path)
	return m.addNewErr[path]
}

func (m *mockChezmoiRunner) Apply(_ context.Context, path string) error {
	m.applyCalls = append(m.applyCalls, path)
	return m.applyErr[path]
}

func (m *mockChezmoiRunner) ApplyAll(_ context.Context) error {
	m.applyAllCalled = true
	return m.applyAllErr
}

func (m *mockChezmoiRunner) Forget(_ context.Context, path string) error {
	m.forgetCalls = append(m.forgetCalls, path)
	return m.forgetErr[path]
}

func (m *mockChezmoiRunner) SourcePath() string {
	return m.sourcePath
}

// mockGitRunner implements git.Runner for testing.
type mockGitRunner struct {
	statusEntries    []git.StatusEntry
	statusErr        error
	diffOutput       map[string]string
	diffErr          map[string]error
	addErr           map[string]error
	addAllErr        error
	commitErr        error
	pushErr          error
	pullErr          error
	resetErr         map[string]error
	restoreErr       map[string]error
	cleanErr         map[string]error
	aheadBehindInfo  git.AheadBehindInfo
	aheadBehindErr   error

	addCalls           []string
	addAllCalled       bool
	commitCalls        []string
	pushCalled         bool
	pullCalled         bool
	resetCalls         []string
	restoreCalls       []string
	cleanCalls         []string
	aheadBehindCalled  bool
}

func newMockGit() *mockGitRunner {
	return &mockGitRunner{
		diffOutput: make(map[string]string),
		diffErr:    make(map[string]error),
		addErr:     make(map[string]error),
		resetErr:   make(map[string]error),
		restoreErr: make(map[string]error),
		cleanErr:   make(map[string]error),
	}
}

func (m *mockGitRunner) Status(_ context.Context) ([]git.StatusEntry, error) {
	return m.statusEntries, m.statusErr
}

func (m *mockGitRunner) Diff(_ context.Context, path string) (string, error) {
	return m.diffOutput[path], m.diffErr[path]
}

func (m *mockGitRunner) Add(_ context.Context, path string) error {
	m.addCalls = append(m.addCalls, path)
	return m.addErr[path]
}

func (m *mockGitRunner) AddAll(_ context.Context) error {
	m.addAllCalled = true
	return m.addAllErr
}

func (m *mockGitRunner) Commit(_ context.Context, message string) error {
	m.commitCalls = append(m.commitCalls, message)
	return m.commitErr
}

func (m *mockGitRunner) Push(_ context.Context) error {
	m.pushCalled = true
	return m.pushErr
}

func (m *mockGitRunner) Pull(_ context.Context) error {
	m.pullCalled = true
	return m.pullErr
}

func (m *mockGitRunner) Reset(_ context.Context, path string) error {
	m.resetCalls = append(m.resetCalls, path)
	return m.resetErr[path]
}

func (m *mockGitRunner) Restore(_ context.Context, path string) error {
	m.restoreCalls = append(m.restoreCalls, path)
	return m.restoreErr[path]
}

func (m *mockGitRunner) Clean(_ context.Context, path string) error {
	m.cleanCalls = append(m.cleanCalls, path)
	return m.cleanErr[path]
}

func (m *mockGitRunner) AheadBehind(_ context.Context) (git.AheadBehindInfo, error) {
	m.aheadBehindCalled = true
	return m.aheadBehindInfo, m.aheadBehindErr
}

// newTestModel creates a Model with mock runners and dimensions set for testing.
func newTestModel() (Model, *mockChezmoiRunner, *mockGitRunner) {
	cm := newMockChezmoi()
	g := newMockGit()
	m := New(cm, g, "dev")
	m.width = 120
	m.height = 40
	m.updateDimensions()
	return m, cm, g
}
