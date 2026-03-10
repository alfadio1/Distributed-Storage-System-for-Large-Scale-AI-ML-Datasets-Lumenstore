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

	s := grpc.NewServer()
	masterv1.RegisterMasterServiceServer(s, master.NewServer())

	log.Printf("master listening on %s", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("master serve error: %v", err)
	}
}
