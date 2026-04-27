package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/process"
)

type tickMsg time.Time
type clearMsg struct{}

var (
	primary   = lipgloss.Color("#00FFFF")
	secondary = lipgloss.Color("#9D7CFF")
	white     = lipgloss.Color("#FAFAFA")
	muted     = lipgloss.Color("#fdfdfd")
	danger    = lipgloss.Color("#FF4D94")
	bgLight   = lipgloss.Color("#222222")

	headerStyle = lipgloss.NewStyle().Foreground(white).Background(secondary).Padding(0, 2).Bold(true)
	sideStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(muted).Padding(0, 1).Width(12).Foreground(white)
	inspectStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(primary).Background(bgLight).Padding(1).Width(38)
	selStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(primary).Bold(true)
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
	confirmQuit bool 
	search      string
	totalMem    float32
}

func doTick() tea.Cmd {
	return tea.Tick(time.Millisecond*800, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func scanPorts() []portInfo {
	var results []portInfo
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("netstat", "-ano", "-p", "TCP")
	} else {
		cmd = exec.Command("ss", "-tlnp")
	}
	out, _ := cmd.Output()
	lines := strings.Split(string(out), "\n")
	var re *regexp.Regexp
	if runtime.GOOS == "windows" {
		re = regexp.MustCompile(`TCP\s+\d+\.\d+\.\d+\.\d+:(\d+)\s+\d+\.\d+\.\d+\.\d+:\d+\s+LISTENING\s+(\d+)`)
	} else {
		re = regexp.MustCompile(`LISTEN\s+\d+\s+\d+\s+[^:]+:(\d+)\s+[^:]+:\*\s+users:\(\("([^"]+)",pid=(\d+)`)
	}
	for _, line := range lines {
		m := re.FindStringSubmatch(line)
		if len(m) >= 3 {
			var pStr, pidStr, name string
			if runtime.GOOS == "windows" {
				pStr, pidStr = m[1], m[2]
				name = "Ghost"
			} else {
				pStr, name, pidStr = m[1], m[2], m[3]
			}
			cpu, mem := 0.0, float32(0.0)
			if proc, err := process.NewProcess(int32(atoi(pidStr))); err == nil {
				if n, err := proc.Name(); err == nil && (name == "Ghost" || name == "") {
					name = n
				}
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
	m.totalMem = 0
	sLower := strings.ToLower(m.search)
	for _, p := range m.ports {
		if strings.Contains(strings.ToLower(p.name), sLower) || strings.Contains(p.port, m.search) {
			m.filtered = append(m.filtered, p)
			m.totalMem += p.mem
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
		if !m.confirmKill && !m.confirmQuit {
			m.ports = scanPorts()
			m.applyFilter()
		}
		return m, doTick()

	case clearMsg:
		m.lastKill = ""
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "esc" || msg.String() == "n" {
			m.confirmKill = false
			m.confirmQuit = false
			if msg.String() == "esc" {
				m.search = ""
				m.applyFilter()
			}
			return m, nil
		}

		if m.confirmQuit {
			if msg.String() == "y" {
				return m, tea.Quit
			}
			return m, nil
		}

		// Logique standard
		switch msg.String() {
		case "ctrl+c", "q":
			m.confirmQuit = true
			return m, nil

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
				if runtime.GOOS == "windows" {
					exec.Command("taskkill", "/F", "/PID", t.pid).Run()
				} else {
					if p, _ := os.FindProcess(atoi(t.pid)); p != nil { p.Kill() }
				}
				m.lastKill = "KILLED: " + t.name
				m.confirmKill = false
				m.ports = scanPorts()
				m.applyFilter()
				return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg { return clearMsg{} })
			}

		default:
			if len(msg.String()) == 1 && msg.Runes[0] >= 32 && msg.Runes[0] <= 126 {
				m.search += msg.String()
				m.applyFilter()
			}
		}
	}
	return m, nil
}

func renderBar(val float32) string {
	width := 10
	filled := int(val / 100)
	if filled > width { filled = width }
	barStyle := lipgloss.NewStyle().Foreground(primary)
	if val > 300 { barStyle = lipgloss.NewStyle().Foreground(danger) }
	return "[" + barStyle.Render(strings.Repeat("■", filled)+strings.Repeat(" ", width-filled)) + "]"
}

func (m model) View() string {
	if len(m.filtered) == 0 && m.search == "" {
		return "GHOSTPORT // SCANNING..."
	}

	var portsCol, mainCol strings.Builder
	start, end := 0, len(m.filtered)
	if m.cursor > 8 { start = m.cursor - 8 }
	if start+12 < end { end = start + 12 }

	for i := start; i < end; i++ {
		p := m.filtered[i]
		pStr := fmt.Sprintf(" :%-5s ", p.port)
		nStr := fmt.Sprintf(" %-12s %s", p.name, renderBar(p.mem))

		if m.cursor == i {
			portsCol.WriteString(selStyle.Render(pStr) + "\n")
			mainCol.WriteString(selStyle.Render(nStr) + "\n")
		} else {
			portsCol.WriteString(lipgloss.NewStyle().Foreground(white).Render(pStr) + "\n")
			mainCol.WriteString(lipgloss.NewStyle().Foreground(white).Render(nStr) + "\n")
		}
	}

	curr := portInfo{}
	if len(m.filtered) > 0 { curr = m.filtered[m.cursor] }

	inspectText := fmt.Sprintf(
		"UNIT ANALYSIS\n%s\n\nNAME   : %s\nPID    : %s\nPORT   : %s\n\nCPU    : %.2f%%\nMEMORY : %.1f MB\n\n%s",
		strings.Repeat("─", 34),
		curr.name, curr.pid, curr.port, curr.cpu, curr.mem,
		lipgloss.NewStyle().Foreground(danger).Bold(true).Render(m.lastKill),
	)

	if m.confirmKill {
		inspectText += "\n\n" + lipgloss.NewStyle().Background(danger).Foreground(white).Padding(0, 1).Bold(true).Render("KILL PROCESS? (Y/N)")
	} else if m.confirmQuit {
		inspectText += "\n\n" + lipgloss.NewStyle().Background(secondary).Foreground(white).Padding(0, 1).Bold(true).Render("QUIT GHOSTPORT? (Y/N)")
	}

	layout := lipgloss.JoinHorizontal(lipgloss.Top,
		sideStyle.Render(portsCol.String()),
		lipgloss.NewStyle().Width(32).Render(mainCol.String()),
		inspectStyle.Foreground(white).Render(inspectText),
	)

	stats := fmt.Sprintf(" NODES: %d | TOTAL RSS: %.1f MB ", len(m.filtered), m.totalMem)
	header := headerStyle.Render(" GHOSTPORT ENGINE ") + lipgloss.NewStyle().Foreground(muted).Render(stats)
	searchBar := lipgloss.NewStyle().Foreground(primary).Bold(true).Render("\n SEARCH > " + m.search + "_")
	footer := lipgloss.NewStyle().Foreground(muted).Render("\n K: KILL • Q: QUIT • ESC: RESET")

	return lipgloss.NewStyle().Padding(1, 2).Render(header + searchBar + "\n\n" + layout + footer)
}

func main() {
	m := model{ports: scanPorts()}
	m.applyFilter()
	if _, err := tea.NewProgram(&m, tea.WithAltScreen()).Run(); err != nil {
		os.Exit(1)
	}
}
