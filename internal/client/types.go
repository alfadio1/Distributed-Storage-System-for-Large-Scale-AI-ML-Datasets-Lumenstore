package client

type ChunkPlan struct {
	ChunkID          string
	Index            uint32
	PrimaryNode      string
	ReplicaNodes     []string
	PrimaryAddress   string
	ReplicaAddresses []string
}

type ObjectPlan struct {
	ObjectKey         string
	SizeBytes         uint64
	ChunkSizeBytes    uint64
	ReplicationFactor uint32
	NumChunks         uint32
	Chunks            []ChunkPlan
}

type LocalChunk struct {
	ChunkID string
	Index   uint32
	Data    []byte
}
