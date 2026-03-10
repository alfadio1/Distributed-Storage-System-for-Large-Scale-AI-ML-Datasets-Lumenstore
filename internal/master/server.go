package master

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	masterv1 "github.com/alpha/lumenstore/gen/master/v1"
)

type Server struct {
	masterv1.UnimplementedMasterServiceServer

	mu    sync.Mutex
	state *State
}

func NewServer() *Server {
	return &Server{
		state: NewState(),
	}
}

func (s *Server) RegisterNode(ctx context.Context, req *masterv1.RegisterNodeRequest) (*masterv1.RegisterNodeResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.Nodes[req.NodeId] = req.Address
	s.state.LastSeen[req.NodeId] = time.Now()

	log.Printf("[master] registered node=%s addr=%s", req.NodeId, req.Address)
	return &masterv1.RegisterNodeResponse{Ok: true}, nil
}

func (s *Server) Heartbeat(ctx context.Context, req *masterv1.HeartbeatRequest) (*masterv1.HeartbeatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.LastSeen[req.NodeId] = time.Now()
	log.Printf("[master] heartbeat node=%s", req.NodeId)

	return &masterv1.HeartbeatResponse{Ok: true}, nil
}

func (s *Server) PutObjectInit(ctx context.Context, req *masterv1.PutObjectInitRequest) (*masterv1.PutObjectInitResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nodeIDs := sortedNodeIDs(s.state.Nodes)

	objectMeta, err := buildObjectPlan(
		req.ObjectKey,
		req.SizeBytes,
		req.ChunkSizeBytes,
		req.ReplicationFactor,
		nodeIDs,
	)
	if err != nil {
		return nil, err
	}

	s.state.Objects[req.ObjectKey] = objectMeta

	respChunks := make([]*masterv1.ChunkPlacement, 0, len(objectMeta.Chunks))
	for _, ch := range objectMeta.Chunks {
		replicaAddresses := make([]string, 0, len(ch.ReplicaNodes))
		for _, replicaNodeID := range ch.ReplicaNodes {
			replicaAddresses = append(replicaAddresses, s.state.Nodes[replicaNodeID])
		}

		respChunks = append(respChunks, &masterv1.ChunkPlacement{
			ChunkId:          ch.ChunkID,
			Index:            ch.Index,
			PrimaryNode:      ch.PrimaryNode,
			ReplicaNodes:     ch.ReplicaNodes,
			PrimaryAddress:   s.state.Nodes[ch.PrimaryNode],
			ReplicaAddresses: replicaAddresses,
		})
	}

	log.Printf("[master] planned object=%s size=%d chunk_size=%d rf=%d num_chunks=%d",
		objectMeta.ObjectKey,
		objectMeta.SizeBytes,
		objectMeta.ChunkSizeBytes,
		objectMeta.ReplicationFactor,
		objectMeta.NumChunks,
	)

	return &masterv1.PutObjectInitResponse{
		ObjectKey:         objectMeta.ObjectKey,
		SizeBytes:         objectMeta.SizeBytes,
		ChunkSizeBytes:    objectMeta.ChunkSizeBytes,
		ReplicationFactor: objectMeta.ReplicationFactor,
		NumChunks:         objectMeta.NumChunks,
		Chunks:            respChunks,
	}, nil
}

func (s *Server) GetObjectPlan(ctx context.Context, req *masterv1.GetObjectPlanRequest) (*masterv1.GetObjectPlanResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.state.Objects[req.ObjectKey]
	if !ok {
		return nil, fmt.Errorf("object not found: %s", req.ObjectKey)
	}

	respChunks := make([]*masterv1.ChunkPlacement, 0, len(obj.Chunks))
	for _, ch := range obj.Chunks {
		replicaAddresses := make([]string, 0, len(ch.ReplicaNodes))
		for _, replicaNodeID := range ch.ReplicaNodes {
			replicaAddresses = append(replicaAddresses, s.state.Nodes[replicaNodeID])
		}

		respChunks = append(respChunks, &masterv1.ChunkPlacement{
			ChunkId:          ch.ChunkID,
			Index:            ch.Index,
			PrimaryNode:      ch.PrimaryNode,
			ReplicaNodes:     ch.ReplicaNodes,
			PrimaryAddress:   s.state.Nodes[ch.PrimaryNode],
			ReplicaAddresses: replicaAddresses,
		})
	}

	log.Printf("[master] get object plan object=%s num_chunks=%d",
		obj.ObjectKey,
		obj.NumChunks,
	)

	return &masterv1.GetObjectPlanResponse{
		ObjectKey:         obj.ObjectKey,
		SizeBytes:         obj.SizeBytes,
		ChunkSizeBytes:    obj.ChunkSizeBytes,
		ReplicationFactor: obj.ReplicationFactor,
		NumChunks:         obj.NumChunks,
		Chunks:            respChunks,
	}, nil
}

func (s *Server) DebugState(ctx context.Context, req *masterv1.Empty) (*masterv1.DebugStateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	registeredNodes := sortedNodeIDs(s.state.Nodes)

	objectKeys := make([]string, 0, len(s.state.Objects))
	for objectKey := range s.state.Objects {
		objectKeys = append(objectKeys, objectKey)
	}
	sort.Strings(objectKeys)

	objects := make([]*masterv1.DebugObject, 0, len(objectKeys))
	for _, objectKey := range objectKeys {
		obj := s.state.Objects[objectKey]

		chunks := make([]*masterv1.ChunkPlacement, 0, len(obj.Chunks))
		for _, ch := range obj.Chunks {
			replicaAddresses := make([]string, 0, len(ch.ReplicaNodes))
			for _, replicaNodeID := range ch.ReplicaNodes {
				replicaAddresses = append(replicaAddresses, s.state.Nodes[replicaNodeID])
			}

			chunks = append(chunks, &masterv1.ChunkPlacement{
				ChunkId:          ch.ChunkID,
				Index:            ch.Index,
				PrimaryNode:      ch.PrimaryNode,
				ReplicaNodes:     ch.ReplicaNodes,
				PrimaryAddress:   s.state.Nodes[ch.PrimaryNode],
				ReplicaAddresses: replicaAddresses,
			})
		}

		objects = append(objects, &masterv1.DebugObject{
			ObjectKey:         obj.ObjectKey,
			SizeBytes:         obj.SizeBytes,
			ChunkSizeBytes:    obj.ChunkSizeBytes,
			ReplicationFactor: obj.ReplicationFactor,
			NumChunks:         obj.NumChunks,
			Chunks:            chunks,
		})
	}

	return &masterv1.DebugStateResponse{
		RegisteredNodes: registeredNodes,
		Objects:         objects,
	}, nil
}
