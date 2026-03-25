package node

import (
	"context"
	"log"

	nodev1 "github.com/alpha/lumenstore/gen/node/v1"
	"google.golang.org/grpc"
)

type Server struct {
	nodev1.UnimplementedNodeServiceServer

	nodeID  string
	storage *Storage
}

func NewServer(nodeID string) *Server {
	return &Server{
		nodeID:  nodeID,
		storage: NewStorage("data/" + nodeID),
		//storage: NewStorage("data"),  // node stores chunks under data/, but not ideal conceptually
	}
}

func (s *Server) Health(ctx context.Context, req *nodev1.HealthRequest) (*nodev1.HealthResponse, error) {
	return &nodev1.HealthResponse{
		NodeId: s.nodeID,
		Status: "ok",
	}, nil
}

// --- Chunk RPCs ---

func (s *Server) PutChunk(ctx context.Context, req *nodev1.PutChunkRequest) (*nodev1.PutChunkResponse, error) {
	bytesWritten, err := s.storage.PutChunk(req.ChunkId, req.Data, req.Crc32)
	if err != nil {
		log.Printf("[%s] PutChunk error: %v", s.nodeID, err)
		return nil, err
	}

	log.Printf("[%s] stored chunk=%s bytes=%d", s.nodeID, req.ChunkId, bytesWritten)

	return &nodev1.PutChunkResponse{
		Ok:           true,
		BytesWritten: bytesWritten,
	}, nil
}

func (s *Server) GetChunk(ctx context.Context, req *nodev1.GetChunkRequest) (*nodev1.GetChunkResponse, error) {
	data, crc, bytesRead, err := s.storage.GetChunk(req.ChunkId)
	if err != nil {
		log.Printf("[%s] GetChunk error: %v", s.nodeID, err)
		return nil, err
	}

	log.Printf("[%s] served chunk=%s bytes=%d", s.nodeID, req.ChunkId, bytesRead)

	return &nodev1.GetChunkResponse{
		Data:      data,
		Crc32:     crc,
		BytesRead: bytesRead,
	}, nil
}

func (s *Server) ReplicateChunk(ctx context.Context, req *nodev1.ReplicateChunkRequest) (*nodev1.ReplicateChunkResponse, error) {
	data, crc, _, err := s.storage.GetChunk(req.ChunkId)
	if err != nil {
		log.Printf("[%s] ReplicateChunk read error chunk=%s err=%v", s.nodeID, req.ChunkId, err)
		return nil, err
	}

	conn, err := grpc.Dial(req.TargetAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("[%s] ReplicateChunk dial target=%s err=%v", s.nodeID, req.TargetAddress, err)
		return nil, err
	}
	defer conn.Close()

	client := nodev1.NewNodeServiceClient(conn)

	_, err = client.PutChunk(ctx, &nodev1.PutChunkRequest{
		ChunkId: req.ChunkId,
		Data:    data,
		Crc32:   crc,
	})
	if err != nil {
		log.Printf("[%s] ReplicateChunk put target=%s chunk=%s err=%v", s.nodeID, req.TargetAddress, req.ChunkId, err)
		return nil, err
	}

	log.Printf("[%s] replicated chunk=%s to target=%s", s.nodeID, req.ChunkId, req.TargetAddress)

	return &nodev1.ReplicateChunkResponse{Ok: true}, nil
}
