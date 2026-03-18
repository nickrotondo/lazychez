# lazychez

A terminal UI for managing [chezmoi](https://www.chezmoi.io/) dotfiles with integrated git operations. Think [lazygit](https://github.com/jesseduffield/lazygit) for your dotfiles.

Built with Go and the [Bubbletea](https://github.com/charmbracelet/bubbletea) framework.

## Features

- Browse all chezmoi-managed files with drift detection (source vs destination changes)
- Inline diffs with syntax-highlighted output
- Add, apply, and discard dotfile changes
- Add unmanaged files to chezmoi with fuzzy-filtered file picker
- Forget (unmanage) files from chezmoi
- Full git workflow: stage, commit, push, pull — without leaving the TUI
- Responsive layout: side-by-side (wide) or stacked (narrow terminals)
- Vim-style navigation

## Install

### Homebrew

```bash
brew tap nickrotondo/tap
brew install --cask lazychez
```

### Go

```bash
go install github.com/nickrotondo/lazychez@latest
```

### From source

```bash
git clone https://github.com/nickrotondo/lazychez.git
cd lazychez
go build
./lazychez
```

## Prerequisites

- [chezmoi](https://www.chezmoi.io/install/) must be installed and initialized (`chezmoi init`)
- Git (for the git integration pane)

## Usage

Run `lazychez` from anywhere — it automatically finds your chezmoi source directory.

> [!NOTE]
> **Pro Tip:** Alias `lazychez` to something like `lc` or `chez` for quick launching

### Layout

The UI has three panes:

| Pane                  | Description                                                    |
| --------------------- | -------------------------------------------------------------- |
| **[1] Managed Files** | All chezmoi-managed files with drift indicators                |
| **[2] Source Git**    | Git status of your chezmoi source directory                    |
| **[0] Details/Diff**  | More details for the selected pane or file—usually a diff view |

In wide terminals (85+ columns), the file list and git status stack on the left with the diff on the right. In narrow terminals, all three panes stack vertically.

### Keybindings

#### Navigation

| Key                 | Action               |
| ------------------- | -------------------- |
| `j` / `k`           | Move down / up       |
| `g` / `G`           | Jump to top / bottom |
| `Tab` / `Shift+Tab` | Next / previous pane |
| `←` / `→`           | Cycle between file list and git |
| `0` / `1` / `2`     | Jump to pane         |
| `Esc`               | Back from diff pane  |

#### Managed Files pane

| Key     | Action                                  |
| ------- | --------------------------------------- |
| `Space` | Add file (copy destination → source)    |
| `a`     | Apply file (copy source → destination)  |
| `A`     | Apply all files                         |
| `D`     | Discard drift (revert the changed side) |
| `e`     | Edit source (`chezmoi edit`)            |
| `E`     | Edit destination file                   |
| `+`     | Add unmanaged file (fuzzy file picker)  |
| `x`     | Forget file (remove from chezmoi)       |

#### Git pane

| Key     | Action                              |
| ------- | ----------------------------------- |
| `Space` | Stage / unstage file                |
| `a`     | Stage all files                     |
| `c`     | Commit (opens message input)        |
| `p`     | Pull from remote                    |
| `P`     | Push to remote                      |
| `D`     | Discard changes (with confirmation) |

#### General

| Key | Action              |
| --- | ------------------- |
| `r` | Refresh all panes   |
| `C` | Edit chezmoi config |
| `?` | Toggle help overlay |
| `q` | Quit                |

## How it works

lazychez wraps the `chezmoi` and `git` CLIs. It runs `chezmoi managed`, `chezmoi status`, and `git status` to populate the panes, then uses `chezmoi diff`, `chezmoi add`, `chezmoi apply`, `git add`, `git commit`, etc. for all operations.

Drift detection compares source and destination states:

- **Source edited** — the chezmoi source has changes not yet applied to `~`
- **Dest edited** — a file in `~` was changed outside chezmoi

## License

MIT
