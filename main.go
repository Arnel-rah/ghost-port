package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	ports    []string
	cursor   int
	selected string
}

func initialModel() model {
	return model{
		ports: []string{":8080 - Go API", ":3000 - React", ":5432 - Postgres"},
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < len(m.ports)-1 { m.cursor++ }
		}
	}
	return m, nil
}

func (m model) View() string {
	s := " GhostPort - Exorcise your localhost\n\n"
	for i, port := range m.ports {
		cursor := " "
		if m.cursor == i { cursor = ">" }
		s += fmt.Sprintf("%s %s\n", cursor, port)
	}
	s += "\nPress q to quit.\n"
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
