
package scanner

import (
    "os/exec"
    "regexp"
    "runtime"
    "strings"

    "github.com/shirou/gopsutil/v3/process"
    "yourproject/domain"
)

type PortScanner interface {
    Scan() ([]domain.PortInfo, error)
}

type SystemPortScanner struct {
    winRe   *regexp.Regexp
    linuxRe *regexp.Regexp
}

func NewSystemPortScanner() *SystemPortScanner {
    return &SystemPortScanner{
        winRe:   regexp.MustCompile(`TCP\s+\d+\.\d+\.\d+\.\d+:(\d+)\s+\d+\.\d+\.\d+\.\d+:\d+\s+LISTENING\s+(\d+)`),
        linuxRe: regexp.MustCompile(`LISTEN\s+\d+\s+\d+\s+[^:]+:(\d+)\s+[^:]+:\*\s+users:\(\("([^"]+)",pid=(\d+)`),
    }
}

func (s *SystemPortScanner) Scan() ([]domain.PortInfo, error) {
    cmd := s.buildCommand()
    out, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    return s.parseOutput(string(out)), nil
}

func (s *SystemPortScanner) buildCommand() *exec.Cmd {
    if runtime.GOOS == "windows" {
        return exec.Command("netstat", "-ano", "-p", "TCP")
    }
    return exec.Command("ss", "-tlnp")
}

func (s *SystemPortScanner) parseOutput(output string) []domain.PortInfo {
    var results []domain.PortInfo
    lines := strings.Split(output, "\n")

    for _, line := range lines {
        if info := s.extractPortInfo(line); info != nil {
            s.enrichWithProcessInfo(info)
            results = append(results, *info)
        }
    }

    return results
}

func (s *SystemPortScanner) extractPortInfo(line string) *domain.PortInfo {
    var matches []string
    if runtime.GOOS == "windows" {
        matches = s.winRe.FindStringSubmatch(line)
        if len(matches) >= 3 {
            return &domain.PortInfo{
                Port: matches[1],
                PID:  matches[2],
                Name: "Ghost",
            }
        }
    } else {
        matches = s.linuxRe.FindStringSubmatch(line)
        if len(matches) >= 4 {
            return &domain.PortInfo{
                Port: matches[1],
                Name: matches[2],
                PID:  matches[3],
            }
        }
    }
    return nil
}

func (s *SystemPortScanner) enrichWithProcessInfo(info *domain.PortInfo) {
    proc, err := process.NewProcess(int32(atoi(info.PID)))
    if err != nil {
        return
    }

    if name, err := proc.Name(); err == nil && (info.Name == "Ghost" || info.Name == "") {
        info.Name = name
    }

    info.CPU, _ = proc.CPUPercent()
    if memInfo, _ := proc.MemoryInfo(); memInfo != nil {
        info.Mem = float32(memInfo.RSS) / 1024 / 1024
    }
}

func atoi(s string) int {
    var res int
    fmt.Sscanf(s, "%d", &res)
    return res
}
