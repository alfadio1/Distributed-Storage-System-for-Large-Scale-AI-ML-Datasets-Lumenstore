package client

import (
	"fmt"
	"os"
)

func BuildLocalChunks(filePath string, plan *ObjectPlan) ([]LocalChunk, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if uint64(len(data)) != plan.SizeBytes {
		return nil, fmt.Errorf("file size mismatch: got=%d want=%d", len(data), plan.SizeBytes)
	}

	chunks := make([]LocalChunk, 0, len(plan.Chunks))

	for i, chunkPlan := range plan.Chunks {
		start := uint64(i) * plan.ChunkSizeBytes
		end := start + plan.ChunkSizeBytes
		if end > uint64(len(data)) {
			end = uint64(len(data))
		}

		chunkData := data[start:end]

		chunks = append(chunks, LocalChunk{
			ChunkID: chunkPlan.ChunkID,
			Index:   chunkPlan.Index,
			Data:    chunkData,
		})
	}

	if len(chunks) != int(plan.NumChunks) {
		return nil, fmt.Errorf("chunk count mismatch: got=%d want=%d", len(chunks), plan.NumChunks)
	}

	return chunks, nil
}
