package filter

import (
	"sort"
	"strconv"
	"strings"

	"github.com/Arnel-rah/ghostport/domain"
)

type PortFilter struct {
	searchTerm string
	sortMode   domain.SortMode
}

func NewPortFilter() *PortFilter {
	return &PortFilter{}
}

func (f *PortFilter) SetSearchTerm(term string) {
	f.searchTerm = strings.ToLower(term)
}

func (f *PortFilter) GetSearchTerm() string {
	return f.searchTerm
}

func (f *PortFilter) SetSortMode(mode domain.SortMode) {
	f.sortMode = mode
}

func (f *PortFilter) GetSortMode() domain.SortMode {
	return f.sortMode
}
func (f *PortFilter) Filter(ports []domain.PortInfo) []domain.PortInfo {
	filtered := f.applySearchFilter(ports)
	f.applySort(filtered)
	return filtered
}

func (f *PortFilter) applySearchFilter(ports []domain.PortInfo) []domain.PortInfo {
	if f.searchTerm == "" {
		return ports
	}

	filtered := make([]domain.PortInfo, 0, len(ports))
	for _, p := range ports {
		if strings.Contains(strings.ToLower(p.Name), f.searchTerm) ||
			strings.Contains(strings.ToLower(p.Port), f.searchTerm) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (f *PortFilter) applySort(ports []domain.PortInfo) {
	sort.Slice(ports, func(i, j int) bool {
		switch f.sortMode {
		case domain.SortName:
			return strings.ToLower(ports[i].Name) < strings.ToLower(ports[j].Name)
		case domain.SortRAM:
			return ports[i].Mem > ports[j].Mem
		default:
			return parsePort(ports[i].Port) < parsePort(ports[j].Port)
		}
	})
}

func parsePort(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
