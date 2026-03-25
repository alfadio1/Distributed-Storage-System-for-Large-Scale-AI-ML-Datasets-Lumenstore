package master

import (
	"fmt"
	"sort"
)

// sortedNodeIDs returns registered node IDs in deterministic sorted order.
func sortedNodeIDs(nodes map[string]NodeMeta) []string {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// buildObjectPlan computes chunk placements for an object.
func buildObjectPlan(
	objectKey string,
	sizeBytes uint64,
	chunkSizeBytes uint64,
	replicationFactor uint32,
	nodeIDs []string,
) (ObjectMeta, error) {

	if objectKey == "" {
		return ObjectMeta{}, fmt.Errorf("object_key is required")
	}
	if sizeBytes == 0 {
		return ObjectMeta{}, fmt.Errorf("size_bytes must be > 0")
	}
	if chunkSizeBytes == 0 {
		return ObjectMeta{}, fmt.Errorf("chunk_size_bytes must be > 0")
	}
	if replicationFactor == 0 {
		return ObjectMeta{}, fmt.Errorf("replication_factor must be > 0")
	}
	if len(nodeIDs) == 0 {
		return ObjectMeta{}, fmt.Errorf("no storage nodes registered")
	}
	if int(replicationFactor) > len(nodeIDs) {
		return ObjectMeta{}, fmt.Errorf("replication_factor=%d exceeds available nodes=%d", replicationFactor, len(nodeIDs))
	}

	numChunks := uint32((sizeBytes + chunkSizeBytes - 1) / chunkSizeBytes)
	chunks := make([]ChunkMeta, 0, numChunks)

	for i := uint32(0); i < numChunks; i++ {
		primaryIdx := int(i) % len(nodeIDs)
		primary := nodeIDs[primaryIdx]

		replicas := make([]string, 0, replicationFactor)
		for r := 0; r < int(replicationFactor); r++ {
			nodeIdx := (primaryIdx + r) % len(nodeIDs)
			replicas = append(replicas, nodeIDs[nodeIdx])
		}

		chunkID := fmt.Sprintf("%s:%06d", objectKey, i)

		chunks = append(chunks, ChunkMeta{
			ChunkID:      chunkID,
			Index:        i,
			PrimaryNode:  primary,
			ReplicaNodes: replicas,
		})
	}

	return ObjectMeta{
		ObjectKey:         objectKey,
		SizeBytes:         sizeBytes,
		ChunkSizeBytes:    chunkSizeBytes,
		ReplicationFactor: replicationFactor,
		NumChunks:         numChunks,
		Chunks:            chunks,
	}, nil
}
