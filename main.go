package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/process"
)

type tickMsg time.Time

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).MarginBottom(1)
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Italic(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFD1")).Background(lipgloss.Color("#222222")).PaddingLeft(1)
	instructionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).MarginTop(1)
)

type portInfo struct {
	port string
	pid  string
	name string
}

type model struct {
	ports  []portInfo
	cursor int
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func scanPorts() []portInfo {
	var results []portInfo
	cmd := exec.Command("netstat", "-ano", "-p", "TCP")
	output, _ := cmd.Output()

	lines := strings.Split(string(output), "\n")
	re := regexp.MustCompile(`TCP\s+\d+\.\d+\.\d+\.\d+:(\d+)\s+\d+\.\d+\.\d+\.\d+:\d+\s+LISTENING\s+(\d+)`)

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			port, pid := matches[1], matches[2]
			name := "Unknown"
			if p, err := process.NewProcess(int32(atoi(pid))); err == nil {
				if n, err := p.Name(); err == nil { name = n }
			}
			results = append(results, portInfo{port, pid, name})
		}
	}
	return results
}

func atoi(s string) int {
	var res int
	fmt.Sscanf(s, "%d", &res)
	return res
}

func (m model) Init() tea.Cmd {
	return doTick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.ports = scanPorts()
		return m, doTick()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < len(m.ports)-1 { m.cursor++ }
		case "K":
			if len(m.ports) > 0 {
				target := m.ports[m.cursor]
				if p, err := os.FindProcess(atoi(target.pid)); err == nil { p.Kill() }
				m.ports = scanPorts()
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("GHOSTPORT - LIVE EXORCISM"))
	b.WriteString("\n")
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-10s %-10s %s", "PORT", "PID", "PROCESS")))
	b.WriteString("\n")

	for i, p := range m.ports {
		row := fmt.Sprintf("%-10s %-10s %s", p.port, p.pid, p.name)
		if m.cursor == i {
			b.WriteString(selectedStyle.Render("⚡ " + row))
		} else {
			b.WriteString("   " + row)
		}
		b.WriteRune('\n')
	}

	b.WriteString(instructionStyle.Render("Auto-refreshing every 1s • ↑/↓: move • SHIFT+K: kill • Q: quit"))
	return b.String()
}

func main() {
	p := tea.NewProgram(model{ports: scanPorts()}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
