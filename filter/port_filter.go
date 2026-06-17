package filter

import (
	"fmt"
	"sort"
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
	f.sortFiltered(filtered)
	return filtered
}

func (f *PortFilter) applySearchFilter(ports []domain.PortInfo) []domain.PortInfo {
	if f.searchTerm == "" {
		return ports
	}

	var filtered []domain.PortInfo
	for _, p := range ports {
		if strings.Contains(strings.ToLower(p.Name), f.searchTerm) ||
			strings.Contains(p.Port, f.searchTerm) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (f *PortFilter) sortFiltered(ports []domain.PortInfo) {
	sort.Slice(ports, func(i, j int) bool {
		switch f.sortMode {
		case domain.SortName:
			return strings.ToLower(ports[i].Name) < strings.ToLower(ports[j].Name)
		case domain.SortRAM:
			return ports[i].Mem > ports[j].Mem
		default:
			return atoi(ports[i].Port) < atoi(ports[j].Port)
		}
	})
}

func atoi(s string) int {
	var res int
	fmt.Sscanf(s, "%d", &res)
	return res
}
