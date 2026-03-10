# LumenStore

Lumen = refers to light / illumination (often used in tech to imply speed, clarity, high-throughput).

Store = indicates a storage system.

LumenStore is a distributed file storage system implemented in **Go**, using **gRPC** for service communication and **Docker** for multi-node deployment.

The system stores large objects by splitting them into fixed-size chunks and distributing those chunks across multiple storage nodes. A centralized metadata service manages object metadata and chunk placement.

This architecture enables:

- parallel reads for high-throughput access
- configurable replication for reliability
- distributed storage across multiple nodes
- metadata-driven object retrieval

LumenStore is designed for workloads that require efficient access to large datasets, particularly **AI/ML training pipelines** where GPUs and TPUs continuously stream large volumes of data.

---

# Architecture

LumenStore consists of three main components:

- **Metadata Service (Master)**
- **Storage Nodes**
- **Client**

```
Client
   │
   ▼
Metadata Service
   │
┌──┼───────────┐
▼  ▼           ▼
Node1 Node2   Node3
```

### Metadata Service

The metadata service acts as the control plane of the system.

It maintains mappings between:

```
Object → Chunks → Storage Nodes
```

Responsibilities:

- node registration
- heartbeat monitoring
- object metadata management
- chunk placement
- replication coordination

The metadata service does **not store file data**.

---

### Storage Nodes

Storage nodes store the actual chunk data on disk.

Each node exposes gRPC APIs that allow clients to:

- write chunks
- read chunks
- replicate chunks

Example layout:

```
node1/
   chunkA
   chunkB

node2/
   chunkC

node3/
   chunkD
```

Chunks are persisted on disk and replicated across nodes according to the configured replication factor.

---

### Client

The client interacts with the storage cluster.

Upload workflow:

```
Client
  │
  │ PutObjectInit
  ▼
Metadata Service
  │
  │ Placement Plan
  ▼
Client uploads chunks
  │
  ▼
Storage Nodes
```

Download workflow:

```
Client
  │
  │ GetObjectPlan
  ▼
Metadata Service
  │
  │ Chunk Locations
  ▼
Client fetches chunks in parallel
```

The client performs concurrent chunk transfers to maximize throughput.

---

# Data Model

Files are stored as **objects** identified by an `ObjectKey`.

Example:

```
ObjectKey: datasets/imagenet.tar
SizeBytes: 5GB
ChunkSize: 64MB
NumChunks: 80
```

Objects are divided into fixed-size chunks:

```
imagenet.tar
├── chunk0
├── chunk1
├── chunk2
└── ...
```

Chunks are distributed across storage nodes.

Example placement:

```
chunk0 → node1
chunk1 → node2
chunk2 → node3
chunk3 → node1
```

---

# Replication

Chunks can be replicated across multiple nodes.

Example replication factor = 2:

```
chunk0 → node1, node2
chunk1 → node2, node3
chunk2 → node3, node1
```

Replication ensures data availability if a node fails.

---

# Consistency Model

LumenStore provides **read-after-write consistency**.

Write flow:

1. client uploads chunk to primary node
2. primary replicates chunk to replica nodes
3. primary acknowledges when replication completes
4. client commits the object

After commit, the object becomes visible for reads.

---

# Failure Handling

Storage nodes periodically send **heartbeats** to the metadata service.

```
node1 → heartbeat → master
node2 → heartbeat → master
node3 → heartbeat → master
```

If a node stops responding, it is marked unavailable.

---

# Replica Healing (Day 6)

If a node failure reduces the replication factor, the system automatically restores replication.

Example:

```
node2 fails
chunk1 → node1, node2
```

Healing process:

```
node1 → replicate chunk1 → node3
```

---

# Technology Stack

- Go
- gRPC
- Protocol Buffers
- Docker
- Concurrent worker pools (goroutines)

---

# Use Case

Distributed storage architectures like this are commonly used in:

- machine learning training pipelines
- large-scale data processing
- dataset distribution across GPU clusters

Instead of reading from a single storage server, compute nodes fetch chunks in parallel across multiple storage nodes, increasing throughput and reducing bottlenecks.

---

# Repository Structure

```
cmd/
   master/
   node/
   client/

internal/
   master/
   node/
   client/

proto/
gen/

docker/
docs/
scripts/
```

---

# Operational Features

LumenStore includes mechanisms designed to support reliability, failure recovery, and performance analysis in distributed environments.

### Automatic Replica Healing

The metadata service continuously monitors node health using heartbeats. If a node failure reduces the replication factor of any chunk, the system automatically restores the desired replication level.

Example scenario:

```
node2 fails
chunk1 replicas: node1, node2
```

The metadata service detects that the chunk is under-replicated and selects a healthy node:

```
node1 → replicate chunk1 → node3
```

After replication completes, the chunk once again satisfies the configured replication factor.

This ensures durability and continued data availability during node failures.

---

### Failure Detection

Storage nodes periodically send heartbeat messages to the metadata service.

```
node1 → heartbeat → master
node2 → heartbeat → master
node3 → heartbeat → master
```

If a node stops responding within the configured timeout window, the metadata service marks the node as unavailable and prevents it from being used for new chunk placements.

---

### Performance Benchmarking

LumenStore includes benchmarking capabilities designed to evaluate system throughput and latency under different workloads.

Typical benchmark scenarios include:

- sequential read throughput
- parallel read throughput
- write latency across multiple nodes
- throughput vs replication factor
- throughput vs chunk size

These benchmarks help evaluate how distributed chunk storage improves throughput compared to single-node storage.