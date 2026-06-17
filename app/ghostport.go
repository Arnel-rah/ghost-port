package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Arnel-rah/ghostport/domain"
	"github.com/Arnel-rah/ghostport/filter"
	"github.com/Arnel-rah/ghostport/killer"
	"github.com/Arnel-rah/ghostport/scanner"
	"github.com/Arnel-rah/ghostport/ui"
)

type GhostPortApp struct {
	config     Config
	scanner    scanner.PortScanner
	killer     killer.ProcessKiller
	filter     *filter.PortFilter
	uiRenderer *ui.PortListRenderer
	styles     ui.Styles
	theme      ui.Theme

	ports       []domain.PortInfo
	filtered    []domain.PortInfo
	cursor      int
	statusMsg   string
	confirmKill bool
	confirmQuit bool
	logs        []string
	totalMem    float32
}

type Config struct {
	RefreshInterval time.Duration
	VisibleRows     int
	MaxLogEntries   int
}

func NewDefaultConfig() Config {
	return Config{
		RefreshInterval: 800 * time.Millisecond,
		VisibleRows:     12,
		MaxLogEntries:   3,
	}
}

func NewGhostPortApp(config Config) *GhostPortApp {
	theme := ui.DefaultTheme()
	styles := ui.NewStyles(theme)

	app := &GhostPortApp{
		config:     config,
		scanner:    scanner.NewSystemPortScanner(),
		killer:     killer.NewSystemProcessKiller(),
		filter:     filter.NewPortFilter(),
		uiRenderer: ui.NewPortListRenderer(styles, theme),
		styles:     styles,
		theme:      theme,
		logs:       make([]string, 0),
	}

	app.refreshPorts()
	return app
}

func (app *GhostPortApp) Init() tea.Cmd {
	return app.tickCmd()
}

func (app *GhostPortApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return app.handleTick()

	case clearMsg:
		app.statusMsg = ""
		return app, nil

	case tea.KeyMsg:
		return app.handleKeyPress(msg)
	}

	return app, nil
}

func (app *GhostPortApp) View() string {
	if len(app.filtered) == 0 && app.filter.GetSearchTerm() == "" {
		return "GHOSTPORT // SCANNING..."
	}

	visibleRange := ui.CalculateVisibleRange(app.cursor, len(app.filtered), app.config.VisibleRows)
	portsCol, mainCol := app.uiRenderer.Render(app.filtered, app.cursor, visibleRange)

	curr := domain.PortInfo{}
	if len(app.filtered) > 0 {
		curr = app.filtered[app.cursor]
	}

	msgColor := app.theme.Danger
	if strings.Contains(app.statusMsg, "REFRESHED") || strings.Contains(app.statusMsg, "SORTED") {
		msgColor = app.theme.Success
	}

	logDisplay := strings.Join(app.logs, "\n")
	inspectText := app.buildInspectText(curr, msgColor, logDisplay)

	layout := lipgloss.JoinHorizontal(lipgloss.Top,
		app.styles.Side.Render(portsCol),
		lipgloss.NewStyle().Width(32).Render(mainCol),
		app.styles.Inspect.Foreground(app.theme.White).Render(inspectText),
	)

	stats := fmt.Sprintf(" NODES: %d | MODE: %s ", len(app.filtered), app.getSortName())
	header := app.styles.Header.Render(" GHOSTPORT ENGINE ") + lipgloss.NewStyle().Foreground(app.theme.Muted).Render(stats)
	searchBar := lipgloss.NewStyle().Foreground(app.theme.Primary).Bold(true).Render("\n SEARCH > "+app.filter.GetSearchTerm()+"_")
	footer := lipgloss.NewStyle().Foreground(app.theme.Muted).Render("\n S: SORT • R: REFRESH • K: KILL • Q: QUIT")

	return lipgloss.NewStyle().Padding(1, 2).Render(header + searchBar + "\n\n" + layout + footer)
}

func (app *GhostPortApp) buildInspectText(curr domain.PortInfo, msgColor lipgloss.Color, logDisplay string) string {
	text := fmt.Sprintf(
		"UNIT ANALYSIS\n%s\n\nNAME   : %s\nPID    : %s\nPORT   : %s\n\nCPU    : %.2f%%\nMEMORY : %.1f MB\n\n%s\n\nLOGS:\n%s",
		strings.Repeat("─", 34),
		curr.Name, curr.PID, curr.Port, curr.CPU, curr.Mem,
		lipgloss.NewStyle().Foreground(msgColor).Bold(true).Render(app.statusMsg),
		lipgloss.NewStyle().Foreground(app.theme.Muted).Render(logDisplay),
	)

	if app.confirmKill {
		text += "\n\n" + lipgloss.NewStyle().Background(app.theme.Danger).Foreground(app.theme.White).Padding(0, 1).Bold(true).Render("KILL PROCESS? (Y/N)")
	} else if app.confirmQuit {
		text += "\n\n" + lipgloss.NewStyle().Background(app.theme.Secondary).Foreground(app.theme.White).Padding(0, 1).Bold(true).Render("QUIT GHOSTPORT? (Y/N)")
	}

	return text
}

func (app *GhostPortApp) handleTick() (tea.Model, tea.Cmd) {
	if !app.confirmKill && !app.confirmQuit {
		app.refreshPorts()
	}
	return app, app.tickCmd()
}

func (app *GhostPortApp) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if app.handleEscapeKey(msg) {
		return app, nil
	}

	if app.confirmQuit {
		return app.handleQuitConfirmation(msg)
	}

	return app.handleNormalKey(msg)
}

func (app *GhostPortApp) handleEscapeKey(msg tea.KeyMsg) bool {
	if msg.String() == "esc" || msg.String() == "n" {
		app.confirmKill = false
		app.confirmQuit = false
		if msg.String() == "esc" {
			app.filter.SetSearchTerm("")
			app.applyFilter()
		}
		return true
	}
	return false
}

func (app *GhostPortApp) handleQuitConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "y" {
		return app, tea.Quit
	}
	app.confirmQuit = false
	return app, nil
}

func (app *GhostPortApp) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		app.confirmQuit = true
		return app, nil

	case "s":
		app.cycleSortMode()
		return app, tea.Tick(time.Second*2, func(t time.Time) tea.Msg { return clearMsg{} })

	case "r":
		app.refreshPorts()
		app.statusMsg = "SYSTEM REFRESHED"
		app.addLog("Manual refresh triggered")
		return app, tea.Tick(time.Second*2, func(t time.Time) tea.Msg { return clearMsg{} })

	case "up", "k":
		if app.cursor > 0 {
			app.cursor--
		}
	case "down", "j":
		if app.cursor < len(app.filtered)-1 {
			app.cursor++
		}

	case "backspace":
		if len(app.filter.GetSearchTerm()) > 0 {
			current := app.filter.GetSearchTerm()
			app.filter.SetSearchTerm(current[:len(current)-1])
			app.applyFilter()
		}

	case "K":
		if len(app.filtered) > 0 {
			app.confirmKill = true
		}

	case "y":
		if app.confirmKill && len(app.filtered) > 0 {
			app.killCurrentProcess()
		}

	default:
		if len(msg.String()) == 1 && len(msg.Runes) > 0 && msg.Runes[0] >= 32 && msg.Runes[0] <= 126 {
			current := app.filter.GetSearchTerm()
			app.filter.SetSearchTerm(current + msg.String())
			app.applyFilter()
		}
	}
	return app, nil
}

func (app *GhostPortApp) cycleSortMode() {
	var mode domain.SortMode
	switch app.filter.GetSortMode() {
	case domain.SortPort:
		mode = domain.SortName
	case domain.SortName:
		mode = domain.SortRAM
	default:
		mode = domain.SortPort
	}
	app.filter.SetSortMode(mode)
	app.applyFilter()
	app.statusMsg = "SORTED " + app.getSortName()
	app.addLog("Sort changed to " + app.getSortName())
}

func (app *GhostPortApp) killCurrentProcess() {
	target := app.filtered[app.cursor]
	if err := app.killer.Kill(target); err == nil {
		app.statusMsg = "KILLED: " + target.Name
		app.addLog("Killed process: " + target.Name)
		app.confirmKill = false
		app.refreshPorts()
	}
}

func (app *GhostPortApp) refreshPorts() {
	if ports, err := app.scanner.Scan(); err == nil {
		app.ports = ports
		app.applyFilter()
	}
}

func (app *GhostPortApp) applyFilter() {
	app.filtered = app.filter.Filter(app.ports)
	app.totalMem = 0
	for _, p := range app.filtered {
		app.totalMem += p.Mem
	}

	if len(app.filtered) == 0 {
		app.cursor = 0
	} else if app.cursor >= len(app.filtered) {
		app.cursor = len(app.filtered) - 1
	}
}

func (app *GhostPortApp) getSortName() string {
	switch app.filter.GetSortMode() {
	case domain.SortName:
		return "BY NAME"
	case domain.SortRAM:
		return "BY RAM"
	default:
		return "BY PORT"
	}
}

func (app *GhostPortApp) addLog(msg string) {
	ts := time.Now().Format("15:04:05")
	app.logs = append([]string{fmt.Sprintf("[%s] %s", ts, msg)}, app.logs...)
	if len(app.logs) > app.config.MaxLogEntries {
		app.logs = app.logs[:app.config.MaxLogEntries]
	}
}

func (app *GhostPortApp) tickCmd() tea.Cmd {
	return tea.Tick(app.config.RefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time
type clearMsg struct{}
