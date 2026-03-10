package client

import (
	"context"
	"fmt"
	"hash/crc32"
	"time"

	nodev1 "github.com/alpha/lumenstore/gen/node/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func UploadChunk(primaryAddress string, chunk LocalChunk) error {
	conn, err := grpc.Dial(primaryAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := nodev1.NewNodeServiceClient(conn)

	checksum := crc32.ChecksumIEEE(chunk.Data)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.PutChunk(ctx, &nodev1.PutChunkRequest{
		ChunkId: chunk.ChunkID,
		Data:    chunk.Data,
		Crc32:   checksum,
	})
	if err != nil {
		return err
	}

	if !resp.Ok {
		return fmt.Errorf("PutChunk returned ok=false for chunk %s", chunk.ChunkID)
	}

	if resp.BytesWritten != uint64(len(chunk.Data)) {
		return fmt.Errorf(
			"bytes written mismatch for chunk %s: got=%d want=%d",
			chunk.ChunkID,
			resp.BytesWritten,
			len(chunk.Data),
		)
	}

	return nil
}

func DownloadChunk(nodeAddr string, chunkID string) ([]byte, uint32, error) {
	conn, err := grpc.Dial(nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, 0, err
	}
	defer conn.Close()

	c := nodev1.NewNodeServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.GetChunk(ctx, &nodev1.GetChunkRequest{
		ChunkId: chunkID,
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Data, resp.Crc32, nil
}
