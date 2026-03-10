package master

import "testing"

func TestBuildObjectPlan(t *testing.T) {
	nodeIDs := []string{"node1", "node2", "node3"}

	objectMeta, err := buildObjectPlan(
		"datasets/imagenet-train.bin",
		268435456, // 256 MB
		67108864,  // 64 MB
		2,         // RF=2
		nodeIDs,
	)
	if err != nil {
		t.Fatalf("buildObjectPlan returned error: %v", err)
	}

	if objectMeta.ObjectKey != "datasets/imagenet-train.bin" {
		t.Fatalf("ObjectKey = %q, want %q", objectMeta.ObjectKey, "datasets/imagenet-train.bin")
	}

	if objectMeta.NumChunks != 4 {
		t.Fatalf("NumChunks = %d, want %d", objectMeta.NumChunks, 4)
	}

	if len(objectMeta.Chunks) != 4 {
		t.Fatalf("len(Chunks) = %d, want %d", len(objectMeta.Chunks), 4)
	}

	tests := []struct {
		index        uint32
		chunkID      string
		primary      string
		replicaNodes []string
	}{
		{
			index:        0,
			chunkID:      "datasets/imagenet-train.bin:000000",
			primary:      "node1",
			replicaNodes: []string{"node1", "node2"},
		},
		{
			index:        1,
			chunkID:      "datasets/imagenet-train.bin:000001",
			primary:      "node2",
			replicaNodes: []string{"node2", "node3"},
		},
		{
			index:        2,
			chunkID:      "datasets/imagenet-train.bin:000002",
			primary:      "node3",
			replicaNodes: []string{"node3", "node1"},
		},
		{
			index:        3,
			chunkID:      "datasets/imagenet-train.bin:000003",
			primary:      "node1",
			replicaNodes: []string{"node1", "node2"},
		},
	}

	for i, tt := range tests {
		got := objectMeta.Chunks[i]

		if got.Index != tt.index {
			t.Fatalf("chunk[%d].Index = %d, want %d", i, got.Index, tt.index)
		}
		if got.ChunkID != tt.chunkID {
			t.Fatalf("chunk[%d].ChunkID = %q, want %q", i, got.ChunkID, tt.chunkID)
		}
		if got.PrimaryNode != tt.primary {
			t.Fatalf("chunk[%d].PrimaryNode = %q, want %q", i, got.PrimaryNode, tt.primary)
		}
		if len(got.ReplicaNodes) != len(tt.replicaNodes) {
			t.Fatalf("chunk[%d].ReplicaNodes len = %d, want %d", i, len(got.ReplicaNodes), len(tt.replicaNodes))
		}
		for j := range tt.replicaNodes {
			if got.ReplicaNodes[j] != tt.replicaNodes[j] {
				t.Fatalf("chunk[%d].ReplicaNodes[%d] = %q, want %q", i, j, got.ReplicaNodes[j], tt.replicaNodes[j])
			}
		}
	}
}
func TestBuildObjectPlanInvalidInputs(t *testing.T) {
	nodeIDs := []string{"node1", "node2"}

	tests := []struct {
		name      string
		objectKey string
		size      uint64
		chunkSize uint64
		rf        uint32
		nodes     []string
	}{
		{
			name:      "empty object key",
			objectKey: "",
			size:      100,
			chunkSize: 10,
			rf:        1,
			nodes:     nodeIDs,
		},
		{
			name:      "zero size",
			objectKey: "obj",
			size:      0,
			chunkSize: 10,
			rf:        1,
			nodes:     nodeIDs,
		},
		{
			name:      "zero chunk size",
			objectKey: "obj",
			size:      100,
			chunkSize: 0,
			rf:        1,
			nodes:     nodeIDs,
		},
		{
			name:      "rf zero",
			objectKey: "obj",
			size:      100,
			chunkSize: 10,
			rf:        0,
			nodes:     nodeIDs,
		},
		{
			name:      "rf exceeds nodes",
			objectKey: "obj",
			size:      100,
			chunkSize: 10,
			rf:        3,
			nodes:     nodeIDs,
		},
		{
			name:      "no nodes",
			objectKey: "obj",
			size:      100,
			chunkSize: 10,
			rf:        1,
			nodes:     []string{},
		},
	}

	for _, tt := range tests {
		_, err := buildObjectPlan(
			tt.objectKey,
			tt.size,
			tt.chunkSize,
			tt.rf,
			tt.nodes,
		)

		if err == nil {
			t.Fatalf("expected error for test case %q but got nil", tt.name)
		}
	}
}
