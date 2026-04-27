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
	purple = lipgloss.Color("#7D56F4")
	cyan   = lipgloss.Color("#00FFD1")
	pink   = lipgloss.Color("#FF0055")
	gray   = lipgloss.Color("#3C3C3C")

	windowStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(1).
			Margin(1)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Background(purple).
			Padding(0, 1).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	killLogStyle = lipgloss.NewStyle().
			Foreground(pink).
			Italic(true)
)

type portInfo struct {
	port string
	pid  string
	name string
}

type model struct {
	ports   []portInfo
	cursor  int
	lastKill string
	width   int
	height  int
}

func doTick() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg { return tickMsg(t) })
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
			name := "Ghost"
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

func (m model) Init() tea.Cmd { return doTick() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		m.ports = scanPorts()
		return m, doTick()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q": return m, tea.Quit
		case "up", "k": if m.cursor > 0 { m.cursor-- }
		case "down", "j": if m.cursor < len(m.ports)-1 { m.cursor++ }
		case "K":
			if len(m.ports) > 0 {
				target := m.ports[m.cursor]
				if p, err := os.FindProcess(atoi(target.pid)); err == nil {
					p.Kill()
					m.lastKill = fmt.Sprintf("EXORCISED: %s (PID %s) at %s", target.name, target.pid, time.Now().Format("15:04:05"))
				}
				m.ports = scanPorts()
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if len(m.ports) == 0 {
		return "Scanning for ghosts..."
	}

	var portList strings.Builder
	for i, p := range m.ports {
		cursor := "  "
		row := fmt.Sprintf("%-6s | %-15s", p.port, p.name)
		if m.cursor == i {
			cursor = "󰚔 "
			portList.WriteString(selectedStyle.Render(cursor + row) + "\n")
		} else {
			portList.WriteString(cursor + row + "\n")
		}
	}

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(40).Render(portList.String()),
		lipgloss.NewStyle().PaddingLeft(4).Render(
			fmt.Sprintf("%s\n\nPID: %s\nStatus: LISTENING\n\n%s",
				titleStyle.Render("SCANNER_DETAILS"),
				m.ports[m.cursor].pid,
				killLogStyle.Render(m.lastKill)),
		),
	)

	footer := lipgloss.NewStyle().Foreground(gray).Render("\n[SHIFT+K] EXORCISE • [Q] DISCONNECT")

	return windowStyle.Render(
		titleStyle.Render(" GHOSTPORT v1.0 - PARANORMAL ACTIVITY DETECTOR ") + "\n\n" +
		mainContent +
		footer,
	)
}

func main() {
	p := tea.NewProgram(model{ports: scanPorts()}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Fatal error: %v", err)
		os.Exit(1)
	}
}
