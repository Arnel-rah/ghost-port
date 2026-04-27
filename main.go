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

	windowStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(purple).Padding(1).Margin(1)
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(purple).Padding(0, 1).Bold(true)
	selStyle    = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	infoStyle   = lipgloss.NewStyle().Foreground(pink).Italic(true)
)

type portInfo struct {
	port string
	pid  string
	name string
	cpu  float64
	mem  float32
}

type model struct {
	ports       []portInfo
	filtered    []portInfo
	cursor      int
	lastKill    string
	confirmKill bool
	search      string
}

func doTick() tea.Cmd {
	return tea.Tick(time.Millisecond*800, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func scanPorts() []portInfo {
	var results []portInfo
	cmd := exec.Command("netstat", "-ano", "-p", "TCP")
	output, err := cmd.Output()
	if err != nil {
		return results
	}
	re := regexp.MustCompile(`TCP\s+\d+\.\d+\.\d+\.\d+:(\d+)\s+\d+\.\d+\.\d+\.\d+:\d+\s+LISTENING\s+(\d+)`)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		m := re.FindStringSubmatch(line)
		if len(m) == 3 {
			p, pidStr := m[1], m[2]
			name, cpu, mem := "Unknown", 0.0, float32(0.0)
			if proc, err := process.NewProcess(int32(atoi(pidStr))); err == nil {
				n, _ := proc.Name()
				if n != "" { name = n }
				cpu, _ = proc.CPUPercent()
				mInfo, _ := proc.MemoryInfo()
				if mInfo != nil { mem = float32(mInfo.RSS) / 1024 / 1024 }
			}
			results = append(results, portInfo{p, pidStr, name, cpu, mem})
		}
	}
	return results
}

func atoi(s string) int {
	var res int
	fmt.Sscanf(s, "%d", &res)
	return res
}

func (m *model) applyFilter() {
	m.filtered = []portInfo{}
	searchLower := strings.ToLower(m.search)
	for _, p := range m.ports {
		if strings.Contains(strings.ToLower(p.name), searchLower) || strings.Contains(p.port, m.search) {
			m.filtered = append(m.filtered, p)
		}
	}
	if len(m.filtered) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

func (m model) Init() tea.Cmd { return doTick() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.ports = scanPorts()
		m.applyFilter()
		return m, doTick()
	case tea.KeyMsg:
		s := msg.String()
		switch s {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < len(m.filtered)-1 { m.cursor++ }
		case "backspace":
			if len(m.search) > 0 {
				m.search = m.search[:len(m.search)-1]
				m.applyFilter()
			}
		case "K":
			if len(m.filtered) > 0 { m.confirmKill = true }
		case "y":
			if m.confirmKill && len(m.filtered) > 0 {
				t := m.filtered[m.cursor]
				if p, err := os.FindProcess(atoi(t.pid)); err == nil {
					p.Kill()
					m.lastKill = fmt.Sprintf("EXORCISED: %s", t.name)
				}
				m.confirmKill = false
				m.ports = scanPorts()
				m.applyFilter()
			}
		case "n":
			m.confirmKill = false
		default:
			if len(s) == 1 {
				m.search += s
				m.applyFilter()
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if len(m.filtered) == 0 && m.search != "" {
		return windowStyle.Render(titleStyle.Render(" GHOSTPORT ") + "\n\nNo ghosts matching: " + m.search + "\n\nBACKSPACE to clear")
	}
	if len(m.filtered) == 0 {
		return windowStyle.Render("Scanning for ghosts...")
	}

	var list strings.Builder
	list.WriteString(fmt.Sprintf("Search: %s_\n\n", m.search))

	start, end := 0, len(m.filtered)
	if m.cursor > 10 { start = m.cursor - 10 }
	if start + 15 < end { end = start + 15 }

	for i := start; i < end; i++ {
		p := m.filtered[i]
		line := fmt.Sprintf("%-6s | %-12s | %3.1fMB", p.port, p.name, p.mem)
		if m.cursor == i {
			list.WriteString(selStyle.Render("> "+line) + "\n")
		} else {
			list.WriteString("  " + line + "\n")
		}
	}

	curr := m.filtered[m.cursor]
	right := fmt.Sprintf("%s\n\nPID: %s\nCPU: %.1f%%\nRAM: %.1f MB\n\n%s",
		titleStyle.Render("SYSTEM_STATS"),
		curr.pid, curr.cpu, curr.mem, infoStyle.Render(m.lastKill))

	if m.confirmKill {
		right += "\n\n" + titleStyle.Background(pink).Render("CONFIRM KILL? (Y/N)")
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(42).Render(list.String()),
		lipgloss.NewStyle().PaddingLeft(4).Render(right))

	return windowStyle.Render(titleStyle.Render(" GHOSTPORT v1.1 ") + "\n\n" + body +
		"\n\n" + lipgloss.NewStyle().Foreground(gray).Render("TYPE TO FILTER • K: KILL • Q: QUIT"))
}

func main() {
	m := model{ports: scanPorts()}
	m.applyFilter()
	if _, err := tea.NewProgram(&m, tea.WithAltScreen()).Run(); err != nil {
		os.Exit(1)
	}
}
