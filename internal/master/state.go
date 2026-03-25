package master

import "time"

// ObjectMeta represents a logical object stored in the system.
type ObjectMeta struct {
	ObjectKey         string
	SizeBytes         uint64
	ChunkSizeBytes    uint64
	ReplicationFactor uint32
	NumChunks         uint32
	Chunks            []ChunkMeta
}

// ChunkMeta represents placement metadata for one chunk.
type ChunkMeta struct {
	ChunkID      string
	Index        uint32
	PrimaryNode  string
	ReplicaNodes []string
}

// NodeMeta represents one registered storage node.
type NodeMeta struct {
	NodeID   string
	Address  string
	LastSeen time.Time
	IsAlive  bool
}

// State is the master's in-memory metadata state.
type State struct {
	Nodes   map[string]NodeMeta // nodeID -> node metadata
	Objects map[string]ObjectMeta
}

func NewState() *State {
	return &State{
		Nodes:   make(map[string]NodeMeta),
		Objects: make(map[string]ObjectMeta),
	}
}
