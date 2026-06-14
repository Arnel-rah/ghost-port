package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Arnel-rah/ghostport/domain"
)

type PortListRenderer struct {
	styles Styles
	theme  Theme
}

func NewPortListRenderer(styles Styles, theme Theme) *PortListRenderer {
	return &PortListRenderer{
		styles: styles,
		theme:  theme,
	}
}

func (r *PortListRenderer) Render(ports []domain.PortInfo, cursor int, visibleRange VisibleRange) (string, string) {
	var portsCol, mainCol strings.Builder

	for i := visibleRange.Start; i < visibleRange.End; i++ {
		p := ports[i]
		portStr := fmt.Sprintf(" :%-5s ", p.Port)
		mainStr := r.formatMainColumn(p)

		if cursor == i {
			portsCol.WriteString(r.styles.Selected.Render(portStr) + "\n")
			mainCol.WriteString(r.styles.Selected.Render(mainStr) + "\n")
		} else {
			portsCol.WriteString(lipgloss.NewStyle().Foreground(r.theme.White).Render(portStr) + "\n")
			mainCol.WriteString(lipgloss.NewStyle().Foreground(r.theme.White).Render(mainStr) + "\n")
		}
	}

	return portsCol.String(), mainCol.String()
}

func (r *PortListRenderer) formatMainColumn(p domain.PortInfo) string {
	displayName := p.Name
	if len(displayName) > 15 {
		displayName = displayName[:12] + "..."
	}

	return fmt.Sprintf(" %-15s %s", displayName, r.renderMemoryBar(p.Mem))
}

func (r *PortListRenderer) renderMemoryBar(mem float32) string {
	width := 10
	filled := int(mem / 100)
	if filled > width {
		filled = width
	}

	barStyle := lipgloss.NewStyle().Foreground(r.theme.Primary)
	if mem > 300 {
		barStyle = lipgloss.NewStyle().Foreground(r.theme.Danger)
	}

	return "[" + barStyle.Render(strings.Repeat("■", filled)+strings.Repeat(" ", width-filled)) + "]"
}

type VisibleRange struct {
	Start int
	End   int
}

func CalculateVisibleRange(cursor, total, windowSize int) VisibleRange {
	start, end := 0, total
	if cursor > 8 {
		start = cursor - 8
	}
	if start+windowSize < end {
		end = start + windowSize
	}
	return VisibleRange{Start: start, End: end}
}
