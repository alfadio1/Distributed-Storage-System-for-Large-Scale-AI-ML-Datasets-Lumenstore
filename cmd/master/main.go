package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	masterv1 "github.com/alpha/lumenstore/gen/master/v1"
	"github.com/alpha/lumenstore/internal/master"
)

func main() {
	addr := ":50051"

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("master listen error: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Create master server instance
	masterServer := master.NewServer()

	// Start heartbeat monitoring loop
	masterServer.StartHeartbeatMonitor()
	masterServer.StartHealer()

	// Register service
	masterv1.RegisterMasterServiceServer(grpcServer, masterServer)

	log.Printf("master listening on %s", addr)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("master serve error: %v", err)
	}
}
