package client

import (
	"fmt"
	"os"
	"sync"
)

func GetFile(masterAddr string, objectKey string, outFile string) error {
	plan, err := RequestGetObjectPlan(masterAddr, objectKey)
	if err != nil {
		return err
	}

	if len(plan.Chunks) == 0 {
		return fmt.Errorf("object has no chunks")
	}

	type downloadTask struct {
		index int
		plan  ChunkPlan
	}

	const workerCount = 4

	chunkBuffers := make([][]byte, len(plan.Chunks))
	tasks := make(chan downloadTask, len(plan.Chunks))
	errCh := make(chan error, len(plan.Chunks))

	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for task := range tasks {
				data, err := downloadChunkWithFallback(task.plan)
				if err != nil {
					errCh <- fmt.Errorf("download failed for chunk %s: %w", task.plan.ChunkID, err)
					continue
				}

				chunkBuffers[task.index] = data
			}
		}()
	}

	for i, ch := range plan.Chunks {
		tasks <- downloadTask{
			index: i,
			plan:  ch,
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

	out, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer out.Close()

	for _, buf := range chunkBuffers {
		_, err := out.Write(buf)
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadChunkWithFallback(ch ChunkPlan) ([]byte, error) {
	addresses := make([]string, 0, 1+len(ch.ReplicaAddresses))

	if ch.PrimaryAddress != "" {
		addresses = append(addresses, ch.PrimaryAddress)
	}

	for _, addr := range ch.ReplicaAddresses {
		if addr != "" && addr != ch.PrimaryAddress {
			addresses = append(addresses, addr)
		}
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("no available addresses for chunk %s", ch.ChunkID)
	}

	var lastErr error
	for _, addr := range addresses {
		data, _, err := DownloadChunk(addr, ch.ChunkID)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all download attempts failed for chunk %s: %w", ch.ChunkID, lastErr)
}
