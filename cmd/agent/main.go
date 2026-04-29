package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"google.golang.org/grpc"
	pb "ghost-port/proto"
	"github.com/shirou/gopsutil/v3/process"
)

var (
	winRe   = regexp.MustCompile(`TCP\s+\d+\.\d+\.\d+\.\d+:(\d+)\s+\d+\.\d+\.\d+\.\d+:\d+\s+LISTENING\s+(\d+)`)
	linuxRe = regexp.MustCompile(`LISTEN\s+\d+\s+\d+\s+[^:]+:(\d+)\s+[^:]+:\*\s+users:\(\("([^"]+)",pid=(\d+)`)
)

type server struct {
	pb.UnimplementedPortMonitorServer
}

func atoi(s string) int {
	var res int
	fmt.Sscanf(s, "%d", &res)
	return res
}

func scanPorts() []*pb.PortInfo {
	var results []*pb.PortInfo
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("netstat", "-ano", "-p", "TCP")
	} else {
		cmd = exec.Command("ss", "-tlnp")
	}
	out, _ := cmd.Output()
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		var res []string
		if runtime.GOOS == "windows" {
			res = winRe.FindStringSubmatch(line)
		} else {
			res = linuxRe.FindStringSubmatch(line)
		}

		if len(res) >= 3 {
			var pStr, pidStr, name string
			if runtime.GOOS == "windows" {
				pStr, pidStr = res[1], res[2]
				name = "Ghost"
			} else {
				pStr, name, pidStr = res[1], res[2], res[3]
			}
			cpu, mem := 0.0, float32(0.0)
			if proc, err := process.NewProcess(int32(atoi(pidStr))); err == nil {
				if n, err := proc.Name(); err == nil && (name == "Ghost" || name == "") {
					name = n
				}
				cpu, _ = proc.CPUPercent()
				mInfo, _ := proc.MemoryInfo()
				if mInfo != nil {
					mem = float32(mInfo.RSS) / 1024 / 1024
				}
			}
			results = append(results, &pb.PortInfo{
				Port: pStr,
				Pid:  pidStr,
				Name: name,
				Cpu:  cpu,
				Mem:  mem,
			})
		}
	}
	return results
}

func (s *server) StreamPorts(empty *pb.Empty, stream pb.PortMonitor_StreamPortsServer) error {
	for {
		ports := scanPorts()
		var totalMem float32
		for _, p := range ports {
			totalMem += p.Mem
		}

		err := stream.Send(&pb.PortList{
			Ports:    ports,
			TotalMem: totalMem,
		})
		if err != nil {
			return err
		}
		time.Sleep(800 * time.Millisecond)
	}
}

func (s *server) KillProcess(ctx context.Context, req *pb.PidRequest) (*pb.KillResponse, error) {
	var err error
	if runtime.GOOS == "windows" {
		err = exec.Command("taskkill", "/F", "/PID", req.Pid).Run()
	} else {
		p, _ := os.FindProcess(atoi(req.Pid))
		if p != nil {
			err = p.Kill()
		}
	}

	if err != nil {
		return &pb.KillResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.KillResponse{Success: true, Message: "Process terminated"}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterPortMonitorServer(s, &server{})
	log.Println("GhostPort-Agent is running on port 50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
