package client

import (
	"context"
	"fmt"
	"time"

	masterv1 "github.com/alpha/lumenstore/gen/master/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func RequestPutObjectPlan(
	masterAddr string,
	objectKey string,
	sizeBytes uint64,
	chunkSizeBytes uint64,
	replicationFactor uint32,
) (*ObjectPlan, error) {
	conn, err := grpc.Dial(masterAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial master: %w", err)
	}
	defer conn.Close()

	client := masterv1.NewMasterServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.PutObjectInit(ctx, &masterv1.PutObjectInitRequest{
		ObjectKey:         objectKey,
		SizeBytes:         sizeBytes,
		ChunkSizeBytes:    chunkSizeBytes,
		ReplicationFactor: replicationFactor,
	})
	if err != nil {
		return nil, fmt.Errorf("PutObjectInit RPC failed: %w", err)
	}

	return convertPutPlanResponse(resp), nil
}

func RequestGetObjectPlan(
	masterAddr string,
	objectKey string,
) (*ObjectPlan, error) {
	conn, err := grpc.Dial(masterAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial master: %w", err)
	}
	defer conn.Close()

	client := masterv1.NewMasterServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.GetObjectPlan(ctx, &masterv1.GetObjectPlanRequest{
		ObjectKey: objectKey,
	})
	if err != nil {
		return nil, fmt.Errorf("GetObjectPlan RPC failed: %w", err)
	}

	return convertGetPlanResponse(resp), nil
}

func convertPutPlanResponse(resp *masterv1.PutObjectInitResponse) *ObjectPlan {
	chunks := make([]ChunkPlan, 0, len(resp.Chunks))
	for _, ch := range resp.Chunks {
		chunks = append(chunks, ChunkPlan{
			ChunkID:          ch.ChunkId,
			Index:            ch.Index,
			PrimaryNode:      ch.PrimaryNode,
			ReplicaNodes:     append([]string(nil), ch.ReplicaNodes...),
			PrimaryAddress:   ch.PrimaryAddress,
			ReplicaAddresses: append([]string(nil), ch.ReplicaAddresses...),
		})
	}

	return &ObjectPlan{
		ObjectKey:         resp.ObjectKey,
		SizeBytes:         resp.SizeBytes,
		ChunkSizeBytes:    resp.ChunkSizeBytes,
		ReplicationFactor: resp.ReplicationFactor,
		NumChunks:         resp.NumChunks,
		Chunks:            chunks,
	}
}

func convertGetPlanResponse(resp *masterv1.GetObjectPlanResponse) *ObjectPlan {
	chunks := make([]ChunkPlan, 0, len(resp.Chunks))
	for _, ch := range resp.Chunks {
		chunks = append(chunks, ChunkPlan{
			ChunkID:          ch.ChunkId,
			Index:            ch.Index,
			PrimaryNode:      ch.PrimaryNode,
			ReplicaNodes:     append([]string(nil), ch.ReplicaNodes...),
			PrimaryAddress:   ch.PrimaryAddress,
			ReplicaAddresses: append([]string(nil), ch.ReplicaAddresses...),
		})
	}

	return &ObjectPlan{
		ObjectKey:         resp.ObjectKey,
		SizeBytes:         resp.SizeBytes,
		ChunkSizeBytes:    resp.ChunkSizeBytes,
		ReplicationFactor: resp.ReplicationFactor,
		NumChunks:         resp.NumChunks,
		Chunks:            chunks,
	}
}
