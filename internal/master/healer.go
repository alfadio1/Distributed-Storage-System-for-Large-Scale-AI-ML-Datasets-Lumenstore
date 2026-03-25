package master

import (
	"context"
	"log"
	"time"

	nodev1 "github.com/alpha/lumenstore/gen/node/v1"
	"google.golang.org/grpc"
)

const (
	healerScanTick = 3 * time.Second
)

// StartHealer runs a background loop that scans for under-replicated chunks
// and repairs them by instructing a healthy source node to replicate the chunk
// to a healthy target node that does not already store it.
func (s *Server) StartHealer() {
	ticker := time.NewTicker(healerScanTick)

	go func() {
		defer ticker.Stop()

		for range ticker.C {
			s.runHealerPass()
		}
	}()
}

func (s *Server) runHealerPass() {
	type repairTask struct {
		objectKey     string
		chunkID       string
		sourceNodeID  string
		sourceAddress string
		targetNodeID  string
		targetAddress string
	}

	tasks := make([]repairTask, 0)

	s.mu.Lock()

	for objectKey, obj := range s.state.Objects {
		for _, ch := range obj.Chunks {
			aliveReplicas := make([]string, 0, len(ch.ReplicaNodes))
			for _, nodeID := range ch.ReplicaNodes {
				node, ok := s.state.Nodes[nodeID]
				if ok && node.IsAlive {
					aliveReplicas = append(aliveReplicas, nodeID)
				}
			}

			if uint32(len(aliveReplicas)) >= obj.ReplicationFactor {
				continue
			}

			sourceNodeID := ""
			if len(aliveReplicas) > 0 {
				sourceNodeID = aliveReplicas[0]
			}
			if sourceNodeID == "" {
				log.Printf("[master] healer cannot repair chunk=%s object=%s reason=no alive source replica",
					ch.ChunkID, objectKey)
				continue
			}

			existing := make(map[string]bool, len(ch.ReplicaNodes))
			for _, nodeID := range ch.ReplicaNodes {
				existing[nodeID] = true
			}

			targetNodeID := ""
			for _, nodeID := range sortedNodeIDs(s.state.Nodes) {
				node := s.state.Nodes[nodeID]
				if !node.IsAlive {
					continue
				}
				if existing[nodeID] {
					continue
				}
				targetNodeID = nodeID
				break
			}

			if targetNodeID == "" {
				log.Printf("[master] healer cannot repair chunk=%s object=%s reason=no healthy target node",
					ch.ChunkID, objectKey)
				continue
			}

			tasks = append(tasks, repairTask{
				objectKey:     objectKey,
				chunkID:       ch.ChunkID,
				sourceNodeID:  sourceNodeID,
				sourceAddress: s.state.Nodes[sourceNodeID].Address,
				targetNodeID:  targetNodeID,
				targetAddress: s.state.Nodes[targetNodeID].Address,
			})
		}
	}

	s.mu.Unlock()

	for _, task := range tasks {
		if err := s.replicateChunkToTarget(task); err != nil {
			log.Printf("[master] healer repair failed chunk=%s object=%s source=%s target=%s err=%v",
				task.chunkID, task.objectKey, task.sourceNodeID, task.targetNodeID, err)
			continue
		}

		s.mu.Lock()
		obj := s.state.Objects[task.objectKey]
		for i := range obj.Chunks {
			if obj.Chunks[i].ChunkID != task.chunkID {
				continue
			}

			alreadyPresent := false
			for _, replicaNodeID := range obj.Chunks[i].ReplicaNodes {
				if replicaNodeID == task.targetNodeID {
					alreadyPresent = true
					break
				}
			}

			if !alreadyPresent {
				obj.Chunks[i].ReplicaNodes = append(obj.Chunks[i].ReplicaNodes, task.targetNodeID)
				s.state.Objects[task.objectKey] = obj
				log.Printf("[master] healer repaired chunk=%s object=%s source=%s target=%s",
					task.chunkID, task.objectKey, task.sourceNodeID, task.targetNodeID)
			}
			break
		}
		s.mu.Unlock()
	}
}

func (s *Server) replicateChunkToTarget(task struct {
	objectKey     string
	chunkID       string
	sourceNodeID  string
	sourceAddress string
	targetNodeID  string
	targetAddress string
}) error {
	conn, err := grpc.Dial(task.sourceAddress, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := nodev1.NewNodeServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.ReplicateChunk(ctx, &nodev1.ReplicateChunkRequest{
		ChunkId:       task.chunkID,
		TargetAddress: task.targetAddress,
	})
	return err
}
