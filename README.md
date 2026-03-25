# LumenStore

LumenStore is a distributed object storage system built in **Go**, designed to support **high-throughput access to large datasets**, particularly for **AI/ML workloads**.

The system splits large files into fixed-size chunks, distributes them across multiple storage nodes, and enables **parallel reads**, **replication**, and **automatic recovery from node failures**.

It is designed to model real-world storage systems used in data-intensive environments where compute (GPUs/TPUs) continuously streams large volumes of data.

---

# Why LumenStore

Modern ML pipelines require:

- high-throughput sequential reads  
- distributed data access across compute nodes  
- resilience to partial failures  
- efficient replication and recovery  

LumenStore addresses these by:

- enabling **parallel chunk fetching across nodes**
- supporting **configurable replication (RF=1–3)**
- maintaining **read-after-write consistency**
- automatically **healing under-replicated data after node failures**

---

# Architecture

LumenStore follows a control plane / data plane separation:

- **Metadata Service (Master)** → control plane  
- **Storage Nodes** → data plane  
- **Client** → orchestrates reads/writes 

```
Client
   │
   v
Metadata Service
   │
┌──┼───────────┐
v  v           v
Node1 Node2   Node3
```


---

## Metadata Service (Master)

The metadata service is responsible for:

- node registration and heartbeats  
- object → chunk → node mapping  
- chunk placement planning  
- replication coordination  
- failure detection and healing  

It does **not store actual file data**.

---

## Storage Nodes

Storage nodes:

- store chunks on disk  
- serve chunk read/write requests  
- participate in replication  

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

The client handles all data movement.

Upload workflow:

```
Client
  │
  │ PutObjectInit
  v
Metadata Service
  │
  │ Placement Plan
  v
Client uploads chunks to nodes (parallel)
  │
  v
Storage Nodes
```

Download workflow:

```
Client
  │
  │ GetObjectPlan
  v
Metadata Service
  │
  │ Chunk Locations
  v
Client fetches chunks in parallel (reassembles file)
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
NumChunks: N
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
chunk0 -> node1
chunk1 -> node2
chunk2 -> node3
chunk3 -> node1
```

---

# Replication

Chunks can be replicated across multiple nodes.

Example replication factor = 2:

```
chunk0 -> node1, node2
chunk1 -> node2, node3
chunk2 -> node3, node1
```


Replication ensures availability under node failure.

---

# Consistency Model

LumenStore provides **read-after-write consistency**.

Write flow:

1. client uploads chunk to primary node  
2. primary replicates to replicas  
3. acknowledgment after replication  
4. client commits object  

After commit, the object is available for reads.

---

# Failure Detection & Recovery

## Heartbeats

Storage nodes send periodic heartbeats:

```
node1 -> heartbeat -> master
node2 -> heartbeat -> master
node3 -> heartbeat -> master
```

If a node stops responding, it is marked unavailable.

---

## Automatic Replica Healing

If a node failure reduces the replication factor, the system automatically restores replication.

Example:

```
node2 fails
chunk1 -> node1, node2
```

Healing process:

```
node1 -> replicate chunk1 -> node3
```

This ensures durability and continued availability.

---

# Performance Benchmarking 

LumenStore includes a built-in benchmarking system to evaluate:

- upload throughput  
- download throughput  
- latency per chunk  
- performance under varying concurrency  
- impact of replication factor  

---

## Benchmark Results

| RF | Workers | Upload (MB/s) | Download (MB/s) |
|----|--------|--------------|----------------|
| 1  | 1      | 76.0         | 76.6           |
| 1  | 4      | 103.6        | 147.4          |
| 1  | 16     | 91.2         | 184.0          |
| 2  | 1      | 43.2         | 100.2          |
| 2  | 4      | 64.2         | 142.6          |
| 2  | 16     | 60.3         | 155.4          |
| 3  | 1      | 32.1         | 154.1          |
| 3  | 4      | 47.1         | 185.9          |
| 3  | 16     | 49.9         | 179.8          |

---

## Key Observations

### Parallelism improves throughput

- Increasing worker count significantly improves read performance  
- Example: RF=1 -> 76 MB/s -> 184 MB/s  

This demonstrates effective **parallel chunk retrieval**.

---

### Replication impacts write throughput

- Higher replication factors reduce upload throughput  
- RF=1 -> ~100 MB/s  
- RF=3 -> ~30–50 MB/s  

This reflects **write amplification** in distributed systems.

---

### Replication improves read availability and performance

- Higher RF increases read flexibility and fallback options  
- RF=3 shows strong read throughput even with fewer workers  

---

### Diminishing returns at high concurrency

- Increasing workers beyond optimal levels introduces overhead  
- Performance plateaus or slightly decreases due to:
  - CPU contention  
  - network saturation  
  - disk I/O limits  

---

# Technology Stack

- Go  
- gRPC  
- Protocol Buffers  
- Docker  
- Goroutines (concurrency)  

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
bench/
```


---

# Running the System

### Start cluster
```
docker compose up
```
---

### Upload file
```
client put --master localhost:50051 --file bigfile.bin --object demo/file --chunk-mb 1 --rf 2
```


---

### Download file
```
client get --master localhost:50051 --object demo/file --out output.bin
```

---

### Run benchmark
```
client bench --master localhost:50051 --file bigfile.bin --object demo/bench --chunk-mb 1 --rf 2 --workers 4 --out bench/results/result.json
```

---

# Summary

LumenStore demonstrates core distributed storage system concepts:

- chunk-based object storage  
- parallel data access  
- replication and fault tolerance  
- failure detection and automatic healing  
- performance benchmarking and system analysis  

The system highlights real-world tradeoffs between:

- throughput vs replication  
- concurrency vs system limits  
- availability vs write cost  

---

# Next Steps

Potential improvements:

- erasure coding  
- stronger consistency models (Raft)  
- multi-region replication  
- adaptive chunk sizing  
- load-aware placement strategies  

```
© 2026 - ALPHA DIALLO
```