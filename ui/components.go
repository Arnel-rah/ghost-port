package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Arnel-rah/ghostport/domain"
)

const (
	memBarWidth    = 10
	nameMaxLen     = 15
	nameTruncLen   = 12
	memDangerMB    = 300
	scrollOffset   = 8
)

type VisibleRange struct {
	Start int
	End   int
}

type PortListRenderer struct {
	styles    Styles
	theme     Theme
	normalStyle lipgloss.Style
}

func NewPortListRenderer(styles Styles, theme Theme) *PortListRenderer {
	return &PortListRenderer{
		styles:      styles,
		theme:       theme,
		normalStyle: lipgloss.NewStyle().Foreground(theme.White),
	}
}

func (r *PortListRenderer) Render(ports []domain.PortInfo, cursor int, visibleRange VisibleRange) (string, string) {
	start := clamp(visibleRange.Start, 0, len(ports))
	end := clamp(visibleRange.End, start, len(ports))

	var portsCol, mainCol strings.Builder

	for i := start; i < end; i++ {
		p := ports[i]
		portStr := fmt.Sprintf(" :%-5s ", p.Port)
		mainStr := r.formatMainColumn(p)

		if cursor == i {
			portsCol.WriteString(r.styles.Selected.Render(portStr) + "\n")
			mainCol.WriteString(r.styles.Selected.Render(mainStr) + "\n")
		} else {
			portsCol.WriteString(r.normalStyle.Render(portStr) + "\n")
			mainCol.WriteString(r.normalStyle.Render(mainStr) + "\n")
		}
	}

	return portsCol.String(), mainCol.String()
}

func (r *PortListRenderer) formatMainColumn(p domain.PortInfo) string {
	name := p.Name
	if len(name) > nameMaxLen {
		name = name[:nameTruncLen] + "..."
	}
	return fmt.Sprintf(" %-15s %s", name, r.renderMemoryBar(p.Mem))
}

func (r *PortListRenderer) renderMemoryBar(mem float32) string {
	filled := int(float32(memBarWidth) * mem / memDangerMB)
	filled = clamp(filled, 0, memBarWidth)

	style := lipgloss.NewStyle().Foreground(r.theme.Primary)
	if mem > memDangerMB {
		style = lipgloss.NewStyle().Foreground(r.theme.Danger)
	}

	bar := strings.Repeat("■", filled) + strings.Repeat(" ", memBarWidth-filled)
	return "[" + style.Render(bar) + "]"
}

func CalculateVisibleRange(cursor, total, windowSize int) VisibleRange {
	if total == 0 {
		return VisibleRange{}
	}

	start := cursor - scrollOffset
	if start < 0 {
		start = 0
	}

	end := start + windowSize
	if end > total {
		end = total
		start = end - windowSize
		if start < 0 {
			start = 0
		}
	}

	return VisibleRange{Start: start, End: end}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
