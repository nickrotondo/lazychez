# lazychez

**[lazygit](https://github.com/jesseduffield/lazygit), but for your dotfiles.**

Your `.zshrc` is on three machines and none of them match. Sound familiar? [chezmoi](https://www.chezmoi.io/) solves this — it's a dotfile manager that tracks your config files in a git repo and applies them consistently across every machine you touch. It's powerful, well-designed, and entirely CLI-driven.

lazychez gives chezmoi a proper terminal UI. Browse your managed files, see exactly what's drifted, view diffs, add or apply changes, and push it all to git — without stringing together five commands from memory.

<!-- TODO: add hero GIF here -->

## Why lazychez

- **See drift instantly** — know which files changed at the source, at the destination, or both
- **Inline diffs** — syntax-highlighted, scrollable, right there in your terminal
- **Full git workflow built in** — stage, commit, push, pull without switching tools
- **Fuzzy file picker** — add unmanaged files to chezmoi without typing paths
- **Forget files** — remove files from chezmoi management when you're done with them
- **Responsive layout** — side-by-side on wide terminals, stacked on narrow ones
- **Vim-style navigation** — because of course

## Install

### Homebrew

```bash
brew tap nickrotondo/tap
brew install lazychez
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

- [chezmoi](https://www.chezmoi.io/install/) installed and initialized (`chezmoi init`)
- Git

## Usage

Run `lazychez` from anywhere — it automatically finds your chezmoi source directory.

> [!TIP]
> Alias it to something short like `lc` or `chez`. Your future self will thank you.

### Layout

The UI has three panes:

| Pane | What it shows |
| --- | --- |
| **[1] Managed Files** | All chezmoi-managed files with drift indicators |
| **[2] Source Git** | Git status of your chezmoi source directory |
| **[0] Diff** | Diff view for the selected file |

Wide terminals (100+ columns) get a side-by-side layout — file list and git status on the left, diff on the right. Narrow terminals stack everything vertically.

<!-- TODO: add layout GIF or screenshot here -->

### How it works

lazychez wraps the `chezmoi` and `git` CLIs under the hood. It calls `chezmoi managed`, `chezmoi status`, and `git status` to populate the panes, then delegates to `chezmoi add`, `chezmoi apply`, `git commit`, etc. for every operation.

**Drift detection** compares source and destination states:

- **●  Source edited** — chezmoi source has changes not yet applied to `~`
- **◆  Dest edited** — a file in `~` was changed outside chezmoi

### Keybindings

<details>
<summary><strong>Navigation</strong></summary>

| Key | Action |
| --- | --- |
| `j` / `k` | Move down / up |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+d` / `Ctrl+u` | Half-page down / up |
| `H` / `L` | Previous / next pane |
| `Tab` / `Shift+Tab` | Next / previous pane |
| `←` / `→` | Cycle between file list and git |
| `0` / `1` / `2` | Jump to pane |
| `Esc` | Back from diff pane |

</details>

<details>
<summary><strong>Managed Files pane</strong></summary>

| Key | Action |
| --- | --- |
| `Space` | Add file (copy destination → source) |
| `a` | Apply file (copy source → destination) |
| `A` | Apply all files |
| `D` | Discard drift (revert the changed side) |
| `e` | Edit source (`chezmoi edit`) |
| `E` | Edit destination file |
| `+` | Add unmanaged file (fuzzy file picker) |
| `x` | Forget file (remove from chezmoi) |

</details>

<details>
<summary><strong>Git pane</strong></summary>

| Key | Action |
| --- | --- |
| `Space` | Stage / unstage file |
| `a` | Stage all files |
| `c` | Commit (opens message input) |
| `p` | Pull from remote |
| `P` | Push to remote |
| `D` | Discard changes (with confirmation) |

</details>

<details>
<summary><strong>General</strong></summary>

| Key | Action |
| --- | --- |
| `r` | Refresh all panes |
| `C` | Edit chezmoi config |
| `?` | Toggle help overlay |
| `q` | Quit |

</details>

## Built with

- [Go](https://go.dev/)
- [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) from the [Charm](https://charm.sh/) ecosystem
