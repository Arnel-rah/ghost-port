package killer

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/Arnel-rah/ghostport/domain"
)

type ProcessKiller interface {
	Kill(port domain.PortInfo) error
}

type SystemProcessKiller struct{}

func NewSystemProcessKiller() *SystemProcessKiller {
	return &SystemProcessKiller{}
}

func (k *SystemProcessKiller) Kill(port domain.PortInfo) error {
	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/F", "/PID", port.PID).Run()
	}

	if p, err := os.FindProcess(atoi(port.PID)); err == nil {
		return p.Kill()
	}
	return nil
}

func atoi(s string) int {
	var res int
	fmt.Sscanf(s, "%d", &res)
	return res
}
