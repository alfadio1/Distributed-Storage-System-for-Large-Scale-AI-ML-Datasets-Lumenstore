package client

import (
	"fmt"
	"sync"
)

func PutFile(
	masterAddr string,
	filePath string,
	objectKey string,
	chunkSizeBytes uint64,
	replicationFactor uint32,
) error {

	fileInfo, err := getFileInfo(filePath)
	if err != nil {
		return err
	}

	plan, err := RequestPutObjectPlan(
		masterAddr,
		objectKey,
		uint64(fileInfo.Size()),
		chunkSizeBytes,
		replicationFactor,
	)
	if err != nil {
		return err
	}

	localChunks, err := BuildLocalChunks(filePath, plan)
	if err != nil {
		return err
	}

	if len(localChunks) != len(plan.Chunks) {
		return fmt.Errorf("local chunk count mismatch: got=%d want=%d", len(localChunks), len(plan.Chunks))
	}

	type uploadTask struct {
		chunk     LocalChunk
		chunkPlan ChunkPlan
	}

	const workerCount = 4

	tasks := make(chan uploadTask, len(localChunks))
	errCh := make(chan error, len(localChunks))

	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for task := range tasks {
				if task.chunk.ChunkID != task.chunkPlan.ChunkID {
					errCh <- fmt.Errorf(
						"chunk plan mismatch at index %d: local=%s plan=%s",
						task.chunk.Index,
						task.chunk.ChunkID,
						task.chunkPlan.ChunkID,
					)
					continue
				}

				if err := UploadChunk(task.chunkPlan.PrimaryAddress, task.chunk); err != nil {
					errCh <- fmt.Errorf("upload failed for chunk %s: %w", task.chunk.ChunkID, err)
					continue
				}
			}
		}()
	}

	for i := range localChunks {
		tasks <- uploadTask{
			chunk:     localChunks[i],
			chunkPlan: plan.Chunks[i],
		}
	}
	close(tasks)

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}
