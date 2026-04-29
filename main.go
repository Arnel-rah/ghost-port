package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/process"
)

type tickMsg time.Time
type clearMsg struct{}

type sortMode int

const (
	sortPort sortMode = iota
	sortName
	sortRAM
)

var (
	winRe   = regexp.MustCompile(`TCP\s+\d+\.\d+\.\d+\.\d+:(\d+)\s+\d+\.\d+\.\d+\.\d+:\d+\s+LISTENING\s+(\d+)`)
	linuxRe = regexp.MustCompile(`LISTEN\s+\d+\s+\d+\s+[^:]+:(\d+)\s+[^:]+:\*\s+users:\(\("([^"]+)",pid=(\d+)`)

	primary   = lipgloss.Color("#00FFFF")
	secondary = lipgloss.Color("#9D7CFF")
	white     = lipgloss.Color("#FAFAFA")
	muted     = lipgloss.Color("#f3f3f3")
	danger    = lipgloss.Color("#FF4D94")
	success   = lipgloss.Color("#00FF88")
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
	statusMsg   string
	confirmKill bool
	confirmQuit bool
	search      string
	totalMem    float32
	mode        sortMode
	logs        []string
}

func doTick() tea.Cmd {
	return tea.Tick(time.Millisecond*800, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *model) addLog(msg string) {
	ts := time.Now().Format("15:04:05")
	m.logs = append([]string{fmt.Sprintf("[%s] %s", ts, msg)}, m.logs...)
	if len(m.logs) > 3 {
		m.logs = m.logs[:3]
	}
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

	for _, line := range lines {
		var res []string
		if runtime.GOOS == "windows" {
			res = winRe.FindStringSubmatch(line)
		} else {
			res = linuxRe.FindStringSubmatch(line)
		}

		if len(res) >= 3 {
			var pStr, pidStr, name string
			if runtime.GOOS == "windows" {
				pStr, pidStr = res[1], res[2]
				name = "Ghost"
			} else {
				pStr, name, pidStr = res[1], res[2], res[3]
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

	sort.Slice(m.filtered, func(i, j int) bool {
		switch m.mode {
		case sortName:
			return strings.ToLower(m.filtered[i].name) < strings.ToLower(m.filtered[j].name)
		case sortRAM:
			return m.filtered[i].mem > m.filtered[j].mem
		default:
			return atoi(m.filtered[i].port) < atoi(m.filtered[j].port)
		}
	})

	if len(m.filtered) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

func (m model) getSortName() string {
	switch m.mode {
	case sortName:
		return "BY NAME"
	case sortRAM:
		return "BY RAM"
	default:
		return "BY PORT"
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
		m.statusMsg = ""
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
			m.confirmQuit = false
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.confirmQuit = true
			return m, nil

		case "s":
			m.mode = (m.mode + 1) % 3
			m.applyFilter()
			m.statusMsg = "SORTED " + m.getSortName()
			m.addLog("Sort changed to " + m.getSortName())
			return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg { return clearMsg{} })

		case "r":
			m.ports = scanPorts()
			m.applyFilter()
			m.statusMsg = "SYSTEM REFRESHED"
			m.addLog("Manual refresh triggered")
			return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg { return clearMsg{} })

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
				if runtime.GOOS == "windows" {
					exec.Command("taskkill", "/F", "/PID", t.pid).Run()
				} else {
					if p, _ := os.FindProcess(atoi(t.pid)); p != nil {
						p.Kill()
					}
				}
				m.statusMsg = "KILLED: " + t.name
				m.addLog("Killed process: " + t.name)
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
	if filled > width {
		filled = width
	}
	barStyle := lipgloss.NewStyle().Foreground(primary)
	if val > 300 {
		barStyle = lipgloss.NewStyle().Foreground(danger)
	}
	return "[" + barStyle.Render(strings.Repeat("■", filled)+strings.Repeat(" ", width-filled)) + "]"
}

func (m model) View() string {
	if len(m.filtered) == 0 && m.search == "" {
		return "GHOSTPORT // SCANNING..."
	}

	var portsCol, mainCol strings.Builder
	start, end := 0, len(m.filtered)
	if m.cursor > 8 {
		start = m.cursor - 8
	}
	if start+12 < end {
		end = start + 12
	}

for i := start; i < end; i++ {
    p := m.filtered[i]
    pStr := fmt.Sprintf(" :%-5s ", p.port)

    displayName := p.name
    if len(displayName) > 15 {
        displayName = displayName[:12] + "..."
    }

    nStr := fmt.Sprintf(" %-15s %s", displayName, renderBar(p.mem))

    if m.cursor == i {
        portsCol.WriteString(selStyle.Render(pStr) + "\n")
        mainCol.WriteString(selStyle.Render(nStr) + "\n")
    } else {
        portsCol.WriteString(lipgloss.NewStyle().Foreground(white).Render(pStr) + "\n")
        mainCol.WriteString(lipgloss.NewStyle().Foreground(white).Render(nStr) + "\n")
    }
}

	curr := portInfo{}
	if len(m.filtered) > 0 {
		curr = m.filtered[m.cursor]
	}

	msgColor := danger
	if strings.Contains(m.statusMsg, "REFRESHED") || strings.Contains(m.statusMsg, "SORTED") {
		msgColor = success
	}

	logDisplay := strings.Join(m.logs, "\n")
	inspectText := fmt.Sprintf(
		"UNIT ANALYSIS\n%s\n\nNAME   : %s\nPID    : %s\nPORT   : %s\n\nCPU    : %.2f%%\nMEMORY : %.1f MB\n\n%s\n\nLOGS:\n%s",
		strings.Repeat("─", 34),
		curr.name, curr.pid, curr.port, curr.cpu, curr.mem,
		lipgloss.NewStyle().Foreground(msgColor).Bold(true).Render(m.statusMsg),
		lipgloss.NewStyle().Foreground(muted).Render(logDisplay),
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

	stats := fmt.Sprintf(" NODES: %d | MODE: %s ", len(m.filtered), m.getSortName())
	header := headerStyle.Render(" GHOSTPORT ENGINE ") + lipgloss.NewStyle().Foreground(muted).Render(stats)
	searchBar := lipgloss.NewStyle().Foreground(primary).Bold(true).Render("\n SEARCH > " + m.search + "_")
	footer := lipgloss.NewStyle().Foreground(muted).Render("\n S: SORT • R: REFRESH • K: KILL • Q: QUIT")

	return lipgloss.NewStyle().Padding(1, 2).Render(header + searchBar + "\n\n" + layout + footer)
}

func main() {
	m := model{ports: scanPorts()}
	m.applyFilter()
	if _, err := tea.NewProgram(&m, tea.WithAltScreen()).Run(); err != nil {
		os.Exit(1)
	}
}
