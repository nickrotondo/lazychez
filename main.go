package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nickrotondo/lazychez/internal/chezmoi"
	"github.com/nickrotondo/lazychez/internal/git"
	"github.com/nickrotondo/lazychez/internal/ui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("lazychez " + version)
		return
	}

	chezmoiRunner, err := chezmoi.NewCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	gitRunner := git.NewCLI(chezmoiRunner.SourcePath())

	p := tea.NewProgram(ui.New(chezmoiRunner, gitRunner), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
