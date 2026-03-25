package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/alpha/lumenstore/internal/client"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage:")
		fmt.Println("  client put --master <addr> --file <path> --object <key> --chunk-mb <mb> --rf <n>")
		fmt.Println("  client get --master <addr> --object <key> --out <path>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "put":
		runPut(os.Args[2:])
	case "get":
		runGet(os.Args[2:])
	case "bench":
		runBench(os.Args[2:])
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}

func runPut(args []string) {
	fs := flag.NewFlagSet("put", flag.ExitOnError)

	masterAddr := fs.String("master", "localhost:50051", "master address")
	filePath := fs.String("file", "", "local file path")
	objectKey := fs.String("object", "", "object key")
	chunkMB := fs.Uint64("chunk-mb", 1, "chunk size in MB")
	rf := fs.Uint("rf", 1, "replication factor")

	fs.Parse(args)

	if *filePath == "" {
		log.Fatal("--file is required")
	}
	if *objectKey == "" {
		log.Fatal("--object is required")
	}

	chunkSizeBytes := (*chunkMB) * 1024 * 1024

	err := client.PutFile(
		*masterAddr,
		*filePath,
		*objectKey,
		chunkSizeBytes,
		uint32(*rf),
	)
	if err != nil {
		log.Fatalf("put failed: %v", err)
	}

	fmt.Printf("put succeeded: file=%s object=%s\n", *filePath, *objectKey)
}

func runGet(args []string) {
	fs := flag.NewFlagSet("get", flag.ExitOnError)

	masterAddr := fs.String("master", "localhost:50051", "master address")
	objectKey := fs.String("object", "", "object key")
	outPath := fs.String("out", "", "output file path")

	fs.Parse(args)

	if *objectKey == "" {
		log.Fatal("--object is required")
	}
	if *outPath == "" {
		log.Fatal("--out is required")
	}

	err := client.GetFile(*masterAddr, *objectKey, *outPath)
	if err != nil {
		log.Fatalf("get failed: %v", err)
	}

	fmt.Printf("get succeeded: object=%s out=%s\n", *objectKey, *outPath)
}

func runBench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)

	masterAddr := fs.String("master", "localhost:50051", "master address")
	filePath := fs.String("file", "", "local file path")
	objectKey := fs.String("object", "", "object key")
	chunkMB := fs.Uint64("chunk-mb", 1, "chunk size in MB")
	rf := fs.Uint("rf", 1, "replication factor")
	workers := fs.Int("workers", 4, "number of parallel workers")
	out := fs.String("out", "bench/results/result.json", "output file")

	fs.Parse(args)

	if *filePath == "" {
		log.Fatal("--file is required")
	}
	if *objectKey == "" {
		log.Fatal("--object is required")
	}

	chunkSizeBytes := (*chunkMB) * 1024 * 1024

	err := client.RunBenchmark(
		*masterAddr,
		*filePath,
		*objectKey,
		chunkSizeBytes,
		uint32(*rf),
		*workers,
		*out,
	)
	if err != nil {
		log.Fatalf("benchmark failed: %v", err)
	}
}
