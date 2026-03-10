package client

import masterv1 "github.com/alpha/lumenstore/gen/master/v1"

func ConvertPlan(resp *masterv1.PutObjectInitResponse) ObjectPlan {

	chunks := make([]ChunkPlan, 0, len(resp.Chunks))

	for _, ch := range resp.Chunks {
		chunks = append(chunks, ChunkPlan{
			ChunkID:          ch.ChunkId,
			Index:            ch.Index,
			PrimaryNode:      ch.PrimaryNode,
			ReplicaNodes:     ch.ReplicaNodes,
			PrimaryAddress:   ch.PrimaryAddress,
			ReplicaAddresses: ch.ReplicaAddresses,
		})
	}

	return ObjectPlan{
		ObjectKey:         resp.ObjectKey,
		SizeBytes:         resp.SizeBytes,
		ChunkSizeBytes:    resp.ChunkSizeBytes,
		ReplicationFactor: resp.ReplicationFactor,
		NumChunks:         resp.NumChunks,
		Chunks:            chunks,
	}
}
