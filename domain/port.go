package domain

type PortInfo struct {
	Port string
	PID  string
	Name string
	CPU  float64
	Mem  float32
}

type SortMode int

const (
	SortPort SortMode = iota
	SortName
	SortRAM
)
