package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type BenchmarkResult struct {
	RunID             string           `json:"run_id"`
	Timestamp         string           `json:"timestamp"`
	MasterAddr        string           `json:"master_addr"`
	FilePath          string           `json:"file_path"`
	ObjectKey         string           `json:"object_key"`
	FileSizeBytes     uint64           `json:"file_size_bytes"`
	ChunkSizeBytes    uint64           `json:"chunk_size_bytes"`
	ReplicationFactor uint32           `json:"replication_factor"`
	Workers           int              `json:"workers"`
	Upload            OperationMetrics `json:"upload"`
	Download          OperationMetrics `json:"download"`
}

type OperationMetrics struct {
	Operation          string         `json:"operation"`
	TotalChunks        int            `json:"total_chunks"`
	TotalBytes         uint64         `json:"total_bytes"`
	DurationMs         float64        `json:"duration_ms"`
	ThroughputMBPerSec float64        `json:"throughput_mb_per_sec"`
	ChunkLatency       LatencySummary `json:"chunk_latency"`
}

type LatencySummary struct {
	Count int     `json:"count"`
	MinMs float64 `json:"min_ms"`
	P50Ms float64 `json:"p50_ms"`
	P95Ms float64 `json:"p95_ms"`
	P99Ms float64 `json:"p99_ms"`
	MaxMs float64 `json:"max_ms"`
}

func buildLatencySummary(durations []time.Duration) LatencySummary {
	if len(durations) == 0 {
		return LatencySummary{}
	}

	msValues := make([]float64, 0, len(durations))
	for _, d := range durations {
		msValues = append(msValues, float64(d.Microseconds())/1000.0)
	}

	sort.Float64s(msValues)

	return LatencySummary{
		Count: len(msValues),
		MinMs: msValues[0],
		P50Ms: percentile(msValues, 50),
		P95Ms: percentile(msValues, 95),
		P99Ms: percentile(msValues, 99),
		MaxMs: msValues[len(msValues)-1],
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	rank := (p / 100.0) * float64(len(sorted)-1)
	lower := int(rank)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	weight := rank - float64(lower)
	return sorted[lower] + (sorted[upper]-sorted[lower])*weight
}

func writeBenchmarkResultJSON(path string, result BenchmarkResult) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create benchmark result dir: %w", err)
		}
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal benchmark result: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write benchmark result file: %w", err)
	}

	return nil
}

func RunBenchmark(
	masterAddr string,
	filePath string,
	objectKey string,
	chunkSizeBytes uint64,
	replicationFactor uint32,
	workerCount int,
	outputPath string,
) error {

	fmt.Println("=== Starting Benchmark ===")

	fileInfo, err := getFileInfo(filePath)
	if err != nil {
		return err
	}

	runID := fmt.Sprintf("run-%d", time.Now().Unix())

	// Upload Phase to measure metrics
	fmt.Println("Uploading...")

	uploadMetrics, err := putFileWithMetrics(
		masterAddr,
		filePath,
		objectKey,
		chunkSizeBytes,
		replicationFactor,
		workerCount,
	)
	if err != nil {
		return fmt.Errorf("upload phase failed: %w", err)
	}

	fmt.Printf("Upload Throughput: %.2f MB/s\n", uploadMetrics.ThroughputMBPerSec)

	// Download Phase: download file to measure metrics
	fmt.Println("Downloading...")

	tmpOut := "bench_download.tmp"

	downloadMetrics, err := getFileWithMetrics(
		masterAddr,
		objectKey,
		tmpOut,
		workerCount,
	)
	if err != nil {
		return fmt.Errorf("download phase failed: %w", err)
	}

	fmt.Printf("Download Throughput: %.2f MB/s\n", downloadMetrics.ThroughputMBPerSec)

	//
	_ = os.Remove(tmpOut)

	result := BenchmarkResult{
		RunID:             runID,
		Timestamp:         time.Now().Format(time.RFC3339),
		MasterAddr:        masterAddr,
		FilePath:          filePath,
		ObjectKey:         objectKey,
		FileSizeBytes:     uint64(fileInfo.Size()),
		ChunkSizeBytes:    chunkSizeBytes,
		ReplicationFactor: replicationFactor,
		Workers:           workerCount,
		Upload:            uploadMetrics,
		Download:          downloadMetrics,
	}

	if err := writeBenchmarkResultJSON(outputPath, result); err != nil {
		return err
	}

	fmt.Println("Benchmark completed")
	fmt.Printf("Results written to: %s\n", outputPath)

	return nil
}
