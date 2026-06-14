package ui

import (
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	White     lipgloss.Color
	Muted     lipgloss.Color
	Danger    lipgloss.Color
	Success   lipgloss.Color
	BgLight   lipgloss.Color
}

func DefaultTheme() Theme {
	return Theme{
		Primary:   lipgloss.Color("#00FFFF"),
		Secondary: lipgloss.Color("#9D7CFF"),
		White:     lipgloss.Color("#FAFAFA"),
		Muted:     lipgloss.Color("#f3f3f3"),
		Danger:    lipgloss.Color("#FF4D94"),
		Success:   lipgloss.Color("#00FF88"),
		BgLight:   lipgloss.Color("#222222"),
	}
}

type Styles struct {
	Header   lipgloss.Style
	Side     lipgloss.Style
	Inspect  lipgloss.Style
	Selected lipgloss.Style
}

func NewStyles(theme Theme) Styles {
	return Styles{
		Header: lipgloss.NewStyle().
			Foreground(theme.White).
			Background(theme.Secondary).
			Padding(0, 2).
			Bold(true),
		Side: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(theme.Muted).
			Padding(0, 1).
			Width(12).
			Foreground(theme.White),
		Inspect: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Primary).
			Background(theme.BgLight).
			Padding(1).
			Width(38),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(theme.Primary).
			Bold(true),
	}
}
