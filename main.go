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
	accent = lipgloss.Color("#00FFD1")
	mute   = lipgloss.Color("#555555")
	bg     = lipgloss.Color("#121212")
	warn   = lipgloss.Color("#FF0055")

	sideStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(mute).
			Padding(0, 1).
			Width(12)

	mainStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Width(30)

	inspectStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1A1A1A")).
			Padding(1).
			Width(35).
			Height(15)

	selLine = lipgloss.NewStyle().Foreground(accent).Bold(true)
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
	out, _ := cmd.Output()
	re := regexp.MustCompile(`TCP\s+\d+\.\d+\.\d+\.\d+:(\d+)\s+\d+\.\d+\.\d+\.\d+:\d+\s+LISTENING\s+(\d+)`)
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		m := re.FindStringSubmatch(line)
		if len(m) == 3 {
			pStr, pidStr := m[1], m[2]
			name, cpu, mem := "Ghost", 0.0, float32(0.0)
			if proc, err := process.NewProcess(int32(atoi(pidStr))); err == nil {
				name, _ = proc.Name()
				cpu, _ = proc.CPUPercent()
				mInfo, _ := proc.MemoryInfo()
				if mInfo != nil {
					mem = float32(mInfo.RSS) / 1024 / 1024
				}
			}
			results = append(results, portInfo{pStr, pidStr, name, cpu, mem})
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
	sLower := strings.ToLower(m.search)
	for _, p := range m.ports {
		if strings.Contains(strings.ToLower(p.name), sLower) || strings.Contains(p.port, m.search) {
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
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "backspace":
			if len(m.search) > 0 {
				m.search = m.search[:len(m.search)-1]
				m.applyFilter()
			}
		case "K":
			if len(m.filtered) > 0 {
				m.confirmKill = true
			}
		case "y":
			if m.confirmKill && len(m.filtered) > 0 {
				t := m.filtered[m.cursor]
				if p, _ := os.FindProcess(atoi(t.pid)); p != nil {
					p.Kill()
					m.lastKill = t.name + " Terminated"
				}
				m.confirmKill = false
				m.ports = scanPorts()
				m.applyFilter()
			}
		case "n":
			m.confirmKill = false
		default:
			if len(msg.String()) == 1 {
				m.search += msg.String()
				m.applyFilter()
			}
		}
	}
	return m, nil
}

func makeProgressBar(percent float32) string {
	width := 10
	filled := int(float32(width) * (percent / 500.0))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("■", filled) + strings.Repeat(" ", width-filled) + "]"
}

func (m model) View() string {
	if len(m.filtered) == 0 {
		return "Searching for spectral activity..."
	}

	var portsCol, mainCol strings.Builder
	start, end := 0, len(m.filtered)
	if m.cursor > 8 {
		start = m.cursor - 8
	}
	if start+15 < end {
		end = start + 15
	}

	for i := start; i < end; i++ {
		p := m.filtered[i]
		pStr := fmt.Sprintf(":%-5s", p.port)
		nStr := fmt.Sprintf("%-12s %s", p.name, makeProgressBar(p.mem))

		if m.cursor == i {
			portsCol.WriteString(selLine.Render(pStr) + "\n")
			mainCol.WriteString(selLine.Render(nStr) + "\n")
		} else {
			portsCol.WriteString(pStr + "\n")
			mainCol.WriteString(nStr + "\n")
		}
	}

	curr := m.filtered[m.cursor]
	inspect := fmt.Sprintf(
		"INSPECTOR\n%s\n\nNAME: %s\nPID:  %s\nPORT: %s\n\nCPU:  %.2f%%\nRAM:  %.1f MB\n\n%s",
		strings.Repeat("─", 30),
		curr.name, curr.pid, curr.port, curr.cpu, curr.mem,
		lipgloss.NewStyle().Foreground(warn).Render(m.lastKill),
	)

	if m.confirmKill {
		inspect += "\n\n" + lipgloss.NewStyle().Background(warn).Bold(true).Padding(0, 1).Render("KILL PROCESS? (Y/N)")
	}

	layout := lipgloss.JoinHorizontal(lipgloss.Top,
		sideStyle.Render(portsCol.String()),
		mainStyle.Render(mainCol.String()),
		inspectStyle.Render(inspect),
	)

	header := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("GHOSTPORT // SEARCH: " + m.search + "_")
	footer := lipgloss.NewStyle().Foreground(mute).Render("\nK: Kill • Q: Quit • Nav: ↑/↓")

	return lipgloss.NewStyle().Padding(1).Render(header + "\n\n" + layout + footer)
}

func main() {
	m := model{ports: scanPorts()}
	m.applyFilter()
	if _, err := tea.NewProgram(&m, tea.WithAltScreen()).Run(); err != nil {
		os.Exit(1)
	}
}
