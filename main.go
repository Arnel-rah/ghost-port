package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Arnel-rah/ghostport/app"
)

func main() {
	config := app.NewDefaultConfig()
	ghostApp := app.NewGhostPortApp(config)

	if _, err := tea.NewProgram(ghostApp, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ghostport: erreur fatale: %v\n", err)
		os.Exit(1)
	}
}
