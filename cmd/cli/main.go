package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "ghost-port/proto"
)

type portMsg *pb.PortList
type errorMsg error

var (
	primary   = lipgloss.Color("#00FFFF")
	secondary = lipgloss.Color("#9D7CFF")
	white     = lipgloss.Color("#FAFAFA")
	muted     = lipgloss.Color("#A0A0A0")
	danger    = lipgloss.Color("#FF4D94")
	success   = lipgloss.Color("#00FF88")
	bgLight   = lipgloss.Color("#222222")

	headerStyle = lipgloss.NewStyle().Foreground(white).Background(secondary).Padding(0, 2).Bold(true)
	sideStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(muted).Padding(0, 1).Width(12).Foreground(white)
	inspectStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(primary).Background(bgLight).Padding(1).Width(38)
	selStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(primary).Bold(true)
)

type model struct {
	client      pb.PortMonitorClient
	stream      pb.PortMonitor_StreamPortsClient
	ports       []*pb.PortInfo
	filtered    []*pb.PortInfo
	cursor      int
	statusMsg   string
	confirmKill bool
	confirmQuit bool
	search      string
	totalMem    float32
	mode        int
	logs        []string
}

func waitForData(stream pb.PortMonitor_StreamPortsClient) tea.Cmd {
	return func() tea.Msg {
		data, err := stream.Recv()
		if err != nil {
			return errorMsg(err)
		}
		return portMsg(data)
	}
}

func (m *model) addLog(msg string) {
	ts := time.Now().Format("15:04:05")
	m.logs = append([]string{fmt.Sprintf("[%s] %s", ts, msg)}, m.logs...)
	if len(m.logs) > 3 {
		m.logs = m.logs[:3]
	}
}

func (m *model) applyFilter() {
	m.filtered = []*pb.PortInfo{}
	sLower := strings.ToLower(m.search)
	for _, p := range m.ports {
		if strings.Contains(strings.ToLower(p.Name), sLower) || strings.Contains(p.Port, m.search) {
			m.filtered = append(m.filtered, p)
		}
	}

	sort.Slice(m.filtered, func(i, j int) bool {
		switch m.mode {
		case 1: return strings.ToLower(m.filtered[i].Name) < strings.ToLower(m.filtered[j].Name)
		case 2: return m.filtered[i].Mem > m.filtered[j].Mem
		default: return m.filtered[i].Port < m.filtered[j].Port
		}
	})

	if len(m.filtered) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

func (m model) Init() tea.Cmd {
	return waitForData(m.stream)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case portMsg:
		m.ports = msg.Ports
		m.totalMem = msg.TotalMem
		m.applyFilter()
		return m, waitForData(m.stream)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.confirmQuit = true
		case "y":
			if m.confirmQuit { return m, tea.Quit }
			if m.confirmKill && len(m.filtered) > 0 {
				target := m.filtered[m.cursor]
				_, err := m.client.KillProcess(context.Background(), &pb.PidRequest{Pid: target.Pid})
				if err == nil {
					m.addLog("Killed: " + target.Name)
				}
				m.confirmKill = false
			}
		case "n", "esc":
			m.confirmKill = false
			m.confirmQuit = false
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < len(m.filtered)-1 { m.cursor++ }
		case "s":
			m.mode = (m.mode + 1) % 3
			m.applyFilter()
		case "K":
			m.confirmKill = true
		case "backspace":
			if len(m.search) > 0 {
				m.search = m.search[:len(m.search)-1]
				m.applyFilter()
			}
		default:
			if len(msg.String()) == 1 {
				m.search += msg.String()
				m.applyFilter()
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	// ... (Garde ta logique View précédente, adapte juste les accès aux champs p.Port, p.Name etc.)
    return "Interface de monitoring gRPC active..." // Simplifié pour l'exemple
}

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewPortMonitorClient(conn)
	stream, err := client.StreamPorts(context.Background(), &pb.Empty{})
	if err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(&model{
		client: client,
		stream: stream,
	}, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
