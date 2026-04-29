package main

import (
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	pb "ghost-port/proto"
)

type server struct {
	pb.UnimplementedPortMonitorServer
}

func (s *server) WatchPorts(req *pb.WatchRequest, stream pb.PortMonitor_WatchPortsServer) error {
	for {
		ports := scanPorts()
		msg := &pb.PortList{
			Ports:    convertPorts(ports),
			TotalMem: 1024.0,
		}

		if err := stream.Send(msg); err != nil {
			return err
		}
		time.Sleep(800 * time.Millisecond)
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterPortMonitorServer(s, &server{})
	log.Println("Agent démarré sur le port 50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
