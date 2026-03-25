package client

import (
	"fmt"
	"sync"
	"time"
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

				// Currently only uplading to primary node

				// if err := UploadChunk(task.chunkPlan.PrimaryAddress, task.chunk); err != nil {
				// 	errCh <- fmt.Errorf("upload failed for chunk %s: %w", task.chunk.ChunkID, err)
				// 	continue
				// }

				//  Upload to primary first
				if err := UploadChunk(task.chunkPlan.PrimaryAddress, task.chunk); err != nil {
					errCh <- fmt.Errorf("primary upload failed for chunk %s: %w", task.chunk.ChunkID, err)
					continue
				}

				// then Upload to replicas
				for _, replicaAddr := range task.chunkPlan.ReplicaAddresses {
					// skip if replica == primary (just in case)
					if replicaAddr == task.chunkPlan.PrimaryAddress {
						continue
					}

					if err := UploadChunk(replicaAddr, task.chunk); err != nil {
						errCh <- fmt.Errorf("replica upload failed for chunk %s to %s: %w",
							task.chunk.ChunkID,
							replicaAddr,
							err,
						)
						continue
					}
				}
				//
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

func putFileWithMetrics(
	masterAddr string,
	filePath string,
	objectKey string,
	chunkSizeBytes uint64,
	replicationFactor uint32,
	workerCount int,
) (OperationMetrics, error) {

	startTime := time.Now()

	fileInfo, err := getFileInfo(filePath)
	if err != nil {
		return OperationMetrics{}, err
	}

	plan, err := RequestPutObjectPlan(
		masterAddr,
		objectKey,
		uint64(fileInfo.Size()),
		chunkSizeBytes,
		replicationFactor,
	)
	if err != nil {
		return OperationMetrics{}, err
	}

	localChunks, err := BuildLocalChunks(filePath, plan)
	if err != nil {
		return OperationMetrics{}, err
	}

	type uploadTask struct {
		chunk     LocalChunk
		chunkPlan ChunkPlan
	}

	tasks := make(chan uploadTask, len(localChunks))
	errCh := make(chan error, len(localChunks))

	// collect latency per chunk
	latencyCh := make(chan time.Duration, len(localChunks))

	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for task := range tasks {

				chunkStart := time.Now()

				// uploading to primary
				if err := UploadChunk(task.chunkPlan.PrimaryAddress, task.chunk); err != nil {
					errCh <- fmt.Errorf("primary upload failed for chunk %s: %w", task.chunk.ChunkID, err)
					continue
				}

				// uploading to replicas
				for _, replicaAddr := range task.chunkPlan.ReplicaAddresses {
					if replicaAddr == task.chunkPlan.PrimaryAddress {
						continue
					}

					if err := UploadChunk(replicaAddr, task.chunk); err != nil {
						errCh <- fmt.Errorf("replica upload failed for chunk %s to %s: %w",
							task.chunk.ChunkID,
							replicaAddr,
							err,
						)
						continue
					}
				}

				chunkDuration := time.Since(chunkStart)
				latencyCh <- chunkDuration
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
	close(latencyCh)

	for err := range errCh {
		if err != nil {
			return OperationMetrics{}, err
		}
	}

	// collect latencies
	var durations []time.Duration
	for d := range latencyCh {
		durations = append(durations, d)
	}

	totalDuration := time.Since(startTime)
	totalBytes := uint64(fileInfo.Size())

	throughput := float64(totalBytes) / (1024 * 1024) / totalDuration.Seconds()

	return OperationMetrics{
		Operation:          "upload",
		TotalChunks:        len(localChunks),
		TotalBytes:         totalBytes,
		DurationMs:         float64(totalDuration.Milliseconds()),
		ThroughputMBPerSec: throughput,
		ChunkLatency:       buildLatencySummary(durations),
	}, nil
}
