package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	masterv1 "github.com/alpha/lumenstore/gen/master/v1"
	nodev1 "github.com/alpha/lumenstore/gen/node/v1"
	"github.com/alpha/lumenstore/internal/node"
)

func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing env var %s", k)
	}
	return v
}

func main() {
	nodeID := mustGetenv("NODE_ID")
	masterAddr := mustGetenv("MASTER_ADDR")
	listenAddr := mustGetenv("LISTEN_ADDR")
	selfAddr := mustGetenv("SELF_ADDR")

	// Start gRPC server
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("node listen error: %v", err)
	}

	s := grpc.NewServer()
	nodev1.RegisterNodeServiceServer(s, node.NewServer(nodeID))
	reflection.Register(s)

	go func() {
		log.Printf("[%s] node listening on %s", nodeID, listenAddr)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("[%s] node serve error: %v", nodeID, err)
		}
	}()

	// Connect to master
	conn, err := grpc.Dial(masterAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("[%s] dial master error: %v", nodeID, err)
	}
	defer conn.Close()

	mc := masterv1.NewMasterServiceClient(conn)

	// Register node
	{
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, err := mc.RegisterNode(ctx, &masterv1.RegisterNodeRequest{
			NodeId:  nodeID,
			Address: selfAddr,
		})

		if err != nil {
			log.Fatalf("[%s] register error: %v", nodeID, err)
		}

		log.Printf("[%s] registered with master (%s)", nodeID, masterAddr)
	}

	// Heartbeat loop
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		_, err := mc.Heartbeat(ctx, &masterv1.HeartbeatRequest{
			NodeId: nodeID,
		})

		cancel()

		if err != nil {
			log.Printf("[%s] heartbeat error: %v", nodeID, err)
			continue
		}
	}
}
