# Plan: Complete Chezmoi File Lifecycle

> Source PRD: https://github.com/nickrotondo/lazychez/issues/1

## Architectural decisions

Durable decisions that apply across all phases:

- **Runner interface**: Two new methods on `chezmoi.Runner` — `Unmanaged(ctx) ([]string, error)` and `Forget(ctx, path) error`
- **Overlay modes**: Two new modes — `OverlayAddFile` (fuzzy file picker) and `OverlayConfirmForget` (y/n confirmation)
- **New component**: `AddFileModel` — a self-contained Bubbletea model wrapping `bubbles/list` with custom multi-select tracking
- **New dependency**: `bubbles/list` (already in the `bubbles` module which is a transitive dependency via `textinput` and `viewport`)
- **Messages**: `UnmanagedMsg`, `AddNewFileResultMsg`, `ForgetResultMsg`
- **Commands**: `fetchUnmanaged`, `addNewFile`, `forgetFile`
- **Keybindings**: `+` (file list pane) opens add-file overlay, `x` (file list pane) forgets selected file with confirmation
- **Mock additions**: `Unmanaged` and `Forget` on the chezmoi mock runner, with call tracking and per-path error injection

---

## Phase 1: Forget a managed file

**User stories**: 5, 6, 8, 14

### What to build

A complete forget flow that lets a user remove a file from chezmoi management. When the user presses `x` on a managed file in the file list pane, a confirmation overlay appears (same pattern as the existing git discard confirmation). Pressing `y` runs `chezmoi forget` on that file, updates the status bar, and refreshes all panes. Pressing `n` or `esc` cancels.

This slice cuts through every layer end-to-end: Runner interface addition, CLI implementation, async command, message type, overlay mode, key handler, help overlay text, mock runner, and tests.

### Acceptance criteria

- [x] `Forget(ctx, path)` added to `chezmoi.Runner` interface and CLI implementation
- [x] `x` keypress in file list pane opens `OverlayConfirmForget` showing the file name
- [x] `y` confirms and calls `chezmoi forget` with the correct full path
- [x] `n` / `esc` cancels without calling forget
- [x] Status bar shows success or error after forget completes
- [x] All panes refresh after successful forget (file disappears from managed list)
- [x] Help overlay updated with `x` keybinding under File Actions
- [x] Mock runner updated with `Forget` support (call tracking, error injection)
- [x] Tests cover: confirm flow, cancel flow, error handling, refresh after forget

---

## Phase 2: Add a single new file via unmanaged overlay

**User stories**: 1, 2, 3, 7, 10, 11, 13

### What to build

A fuzzy-filterable overlay for discovering and adding unmanaged files. When the user presses `+` in the file list pane, lazychez fetches `chezmoi unmanaged` and displays the results in a `bubbles/list`-based overlay with built-in fuzzy filtering. The user can type to filter, navigate with j/k, and press Enter to add the selected file via `chezmoi add`. Esc closes the overlay.

This slice introduces the `AddFileModel` component, the `bubbles/list` dependency, and the `Unmanaged` Runner method. Single-select only — multi-select comes in Phase 3.

**Note:** `chezmoi add` prompts interactively by default. Use `--force` to skip the CLI confirmation, since the TUI already handles user intent (same lesson from Phase 1's `chezmoi forget --force`).

### Acceptance criteria

- [x] `Unmanaged(ctx)` added to `chezmoi.Runner` interface and CLI implementation
- [x] `+` keypress in file list pane triggers `fetchUnmanaged` and opens `OverlayAddFile`
- [x] Overlay displays unmanaged files using `bubbles/list` with fuzzy filtering
- [x] User can type to filter, navigate with j/k or arrow keys
- [x] Enter on a file runs `chezmoi add` for that path
- [x] Status bar shows success or error after add
- [x] All panes refresh after successful add (file appears in managed list and git status)
- [x] Esc closes overlay without adding anything
- [x] Empty unmanaged list shows an appropriate message
- [x] Help overlay updated with `+` keybinding under File Actions
- [x] Mock runner updated with `Unmanaged` support
- [x] Tests cover: overlay open/close, filtering, single add, error handling, empty list, refresh after add

---

## Phase 3: Multi-select and batch add

**User stories**: 4, 9, 12, 15

### What to build

Extend the add-file overlay with multi-select capability. Space toggles selection on the current item (shown with a checkmark indicator). Enter runs `chezmoi add` on all selected files. After a batch add completes, the user stays in the overlay — successfully added files are removed from the list so the user can continue adding more. The overlay only closes on Esc.

### Acceptance criteria

- [ ] Space toggles selection on the current item in the add-file overlay
- [ ] Selected items display a visible checkmark or selection indicator
- [ ] Enter with selections runs `chezmoi add` on each selected file
- [ ] Enter with no selections adds the currently focused file (single-select fallback)
- [ ] User stays in the overlay after batch add completes
- [ ] Successfully added files are removed from the unmanaged list in the overlay
- [ ] Status bar shows progress/results of batch add
- [ ] All panes refresh after batch add
- [ ] Tests cover: multi-select toggle, batch add, stay-in-overlay behavior, error on one file in batch
