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
	"google.golang.org/protobuf/types/known/emptypb" // Import nécessaire

	pb "github.com/Arnel-rah/ghostport/proto"
)

type portMsg *pb.PortList
type errorMsg error

var (
	primary      = lipgloss.Color("#00FFFF")
	secondary    = lipgloss.Color("#9D7CFF")
	white        = lipgloss.Color("#FAFAFA")
	muted        = lipgloss.Color("#A0A0A0")
	danger       = lipgloss.Color("#FF4D94")
	bgLight      = lipgloss.Color("#222222")
	headerStyle  = lipgloss.NewStyle().Foreground(white).Background(secondary).Padding(0, 2).Bold(true)
	selStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(primary).Bold(true)
)

type model struct {
	client      pb.PortMonitorClient
	stream      pb.PortMonitor_StreamPortsClient
	ports       []*pb.PortInfo
	filtered    []*pb.PortInfo
	cursor      int
	search      string
	totalMem    float32
	mode        int
	confirmKill bool
	confirmQuit bool
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
				_, _ = m.client.KillProcess(context.Background(), &pb.PidRequest{Pid: target.Pid})
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
	var s strings.Builder
	s.WriteString(headerStyle.Render(" GHOSTPORT REMOTE MONITOR ") + "\n\n")

	if m.confirmQuit {
		return s.String() + danger.Render(" Quitter l'application ? (y/n)")
	}
	if m.confirmKill {
		return s.String() + danger.Render(" Tuer le processus sélectionné ? (y/n)")
	}

	s.WriteString(fmt.Sprintf(" Filtrer: [%s] | Mode: %d | Total RAM: %.2f MB\n\n", m.search, m.mode, m.totalMem))

	for i, p := range m.filtered {
		cursor := "  "
		lineStyle := lipgloss.NewStyle().Foreground(white)

		// Tronquer le nom pour éviter le dépassement
		displayName := p.Name
		if len(displayName) > 20 {
			displayName = displayName[:17] + "..."
		}

		content := fmt.Sprintf(":%-6s %-20s [%.2f MB]", p.Port, displayName, p.Mem)

		if m.cursor == i {
			cursor = "> "
			s.WriteString(selStyle.Render(cursor+content) + "\n")
		} else {
			s.WriteString(lineStyle.Render(cursor+content) + "\n")
		}
	}

	s.WriteString("\n" + muted.Render("↑/↓: Naviguer | s: Trier | K: Kill | q: Quitter"))
	return s.String()
}

func main() {
	// Connexion à l'Agent
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewPortMonitorClient(conn)
	stream, err := client.StreamPorts(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(&model{
		client: client,
		stream: stream,
	}, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Erreur: %v", err)
		os.Exit(1)
	}
}
